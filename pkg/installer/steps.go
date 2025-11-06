package installer

import (
	"SophonClientv2/internal/logging"
	"SophonClientv2/pkg/utils"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (inst *Installer) EnqueueChunks() {
	// Subscribe to Input channel and enqueue chunks for processing

	orderedChunks := inst.EnumerateChunksWithFileOrder()
	if len(orderedChunks) != len(inst.ChunkMap) {
		logging.GlobalLogger.Fatal("Assertion Failed. Chunk enumeration mismatch. Something is wrong with the code.")
		return
	}

	inst.wg.Add(1)
	go func() {
		defer inst.wg.Done()
		for _, cm := range orderedChunks {
			utils.NonBlockingEnqueue(inst.InputQueue, ChunksInput{Metadata: cm})
		}
		logging.GlobalLogger.Info("All initial chunks enqueued")
	}()
}

func (inst *Installer) DownloadChunks() {
	logging.GlobalLogger.Info("Starting chunk download")

	inst.wg.Add(1)
	go func() {
		defer inst.wg.Done()
		for input := range inst.InputQueue {
			inst.Downloader.EnqueueDownload(input.Metadata.URL, input.Metadata)
		}
		logging.GlobalLogger.Info("InputQueue closed, stopping Downloader")
		inst.Downloader.Stop()
	}()
}

func (inst *Installer) DecompressChunks() {
	logging.GlobalLogger.Info("Starting chunk decompression")

	inst.wg.Add(1)
	go func() {
		defer inst.wg.Done()
		for downloadOutput := range inst.Downloader.GetOutputChannel() {
			cm := downloadOutput.Payload.(*ChunkMetaData)

			if !downloadOutput.Suceeded {
				logging.GlobalLogger.Warn(fmt.Sprintf("Download failed for chunk %s, re-enqueueing", cm.ChunkID))
				utils.NonBlockingEnqueue(inst.InputQueue, ChunksInput{Metadata: cm})
				continue
			}

			if cm.IsCompressed {
				inst.Decompressor.EnqueueDecompression(downloadOutput.Content, cm)

				inst.Progress.mu.Lock()
				inst.Progress.DownloadedBytes += int64(cm.CompressedSize)
				inst.Progress.DownloadedChunks++
				inst.Progress.mu.Unlock()
			} else {
				// TODO: Handle uncompressed chunk (passthrough)
				logging.GlobalLogger.Fatal("Uncompressed chunks are not yet supported")
				return
			}
		}
		logging.GlobalLogger.Info("Downloader output closed, stopping Decompressor")
		inst.Decompressor.Stop()
	}()
}

func (inst *Installer) VerifyChunks() {
	logging.GlobalLogger.Info("Starting chunk verification")

	inst.wg.Add(1)
	go func() {
		defer inst.wg.Done()
		for decompressOutput := range inst.Decompressor.GetOutputChannel() {
			cm := decompressOutput.Payload.(*ChunkMetaData)

			if !decompressOutput.Suceeded {
				logging.GlobalLogger.Warn(fmt.Sprintf("Decompression failed for chunk %s, re-enqueueing", cm.ChunkID))
				utils.NonBlockingEnqueue(inst.InputQueue, ChunksInput{Metadata: cm})

				// Adjust downloaded bytes since we are re-enqueueing
				inst.Progress.mu.Lock()
				inst.Progress.TotalBytes += int64(cm.CompressedSize)
				inst.Progress.mu.Unlock()
				continue
			}

			inst.Verifier.EnqueueVerification(cm.ChunkID, decompressOutput.Content, cm.MD5, cm)

			inst.Progress.mu.Lock()
			inst.Progress.DecompressedChunks++
			inst.Progress.mu.Unlock()
		}
		logging.GlobalLogger.Info("Decompressor output closed, stopping Verifier")
		inst.Verifier.Stop()
	}()
}

func (inst *Installer) AssembleChunks() {
	logging.GlobalLogger.Info("Starting chunk assembly")

	inst.wg.Add(1)
	go func() {
		defer inst.wg.Done()
		for verifyOutput := range inst.Verifier.GetOutputChannel() {
			cm := verifyOutput.Payload.(*ChunkMetaData)

			if !verifyOutput.Suceeded {
				logging.GlobalLogger.Warn(fmt.Sprintf("Verification failed for chunk %s, re-enqueueing", cm.ChunkID))
				utils.NonBlockingEnqueue(inst.InputQueue, ChunksInput{Metadata: cm})

				// Adjust downloaded bytes since we are re-enqueueing
				inst.Progress.mu.Lock()
				inst.Progress.TotalBytes += int64(cm.CompressedSize)
				inst.Progress.mu.Unlock()
				continue
			}

			inst.Progress.mu.Lock()
			inst.Progress.VerifiedChunks++
			inst.Progress.mu.Unlock()

			// This is here because one chunk can be used for multiple files.
			// And readcloser can only be read once.
			// Read content into memory once for reuse across multiple destinations
			contentBytes, err := io.ReadAll(verifyOutput.Content)
			if cerr := verifyOutput.Content.Close(); cerr != nil {
				logging.GlobalLogger.Warn(fmt.Sprintf("Failed to close verified content stream for chunk %s: %v", cm.ChunkID, cerr))
			}

			if err != nil {
				logging.GlobalLogger.Error(fmt.Sprintf("Failed to read verified content for chunk %s: %v, re-enqueueing", cm.ChunkID, err))
				utils.NonBlockingEnqueue(inst.InputQueue, ChunksInput{Metadata: cm})

				inst.Progress.mu.Lock()
				inst.Progress.TotalBytes += int64(cm.CompressedSize)
				inst.Progress.mu.Unlock()
				continue
			}

			// Create a new reader for each destination
			for _, dest := range cm.Destinations {
				inst.Assembler.EnqueueWrite(dest.File.FilePath, dest.Offset, cm.ChunkID, io.NopCloser(bytes.NewReader(contentBytes)), cm)
			}
		}
		logging.GlobalLogger.Info("Verifier output closed, stopping Assembler")
		inst.Assembler.Stop()
	}()
}

func (inst *Installer) VerifyFiles() {
	logging.GlobalLogger.Info("Starting file verification")

	inst.wg.Add(1)
	go func() {
		defer inst.wg.Done()
		// Track which chunk instances (chunkID+offset) have been assembled for each file
		// Key: filePath, Value: map of "chunkID:offset" -> bool
		fileAssembledChunks := make(map[string]map[string]bool)
		// Cache expected chunk count per file to avoid recomputing
		fileExpectedChunks := make(map[string]int)

		for assemblerOutput := range inst.Assembler.GetOutputChannel() {
			cm := assemblerOutput.Payload.(*ChunkMetaData)
			filePath := assemblerOutput.FilePath

			if !assemblerOutput.Succeeded {
				logging.GlobalLogger.Warn(fmt.Sprintf("Assembly failed for chunk %s, re-enqueueing", cm.ChunkID))
				utils.NonBlockingEnqueue(inst.InputQueue, ChunksInput{Metadata: cm})

				// Adjust downloaded bytes since we are re-enqueueing
				inst.Progress.mu.Lock()
				inst.Progress.TotalBytes += int64(cm.CompressedSize)
				inst.Progress.mu.Unlock()
				continue
			}

			inst.Progress.mu.Lock()
			inst.Progress.AssembledChunks++
			inst.Progress.mu.Unlock()

			if fileAssembledChunks[filePath] == nil {
				fileAssembledChunks[filePath] = make(map[string]bool)
			}

			// Find the offset for this specific file to create a unique key
			var offset uint64
			var fileMeta *FileMetaData
			for _, dest := range cm.Destinations {
				if dest.File.FilePath == filePath {
					fileMeta = dest.File
					offset = dest.Offset
					break
				}
			}
			if fileMeta == nil {
				logging.GlobalLogger.Fatal(fmt.Sprintf("File metadata not found for assembled file: %s", filePath))
				return
			}

			// Compute expected chunk count only once per file
			if _, exists := fileExpectedChunks[filePath]; !exists {
				chunkSet := make(map[string]bool)
				for _, chunkID := range fileMeta.Chunks {
					chunkSet[chunkID] = true
				}
				fileExpectedChunks[filePath] = len(chunkSet)
			}

			// Create a unique key for this chunk instance (chunkID:offset)
			chunkInstanceKey := fmt.Sprintf("%s:%d", cm.ChunkID, offset)
			fileAssembledChunks[filePath][chunkInstanceKey] = true

			expectedChunkInstances := fileExpectedChunks[filePath]
			logging.GlobalLogger.Debug(fmt.Sprintf("File %s: Assembled Chunks: %d, Expected Chunk Instances: %d", filePath, len(fileAssembledChunks[filePath]), expectedChunkInstances))
			if len(fileAssembledChunks[filePath]) == expectedChunkInstances {
				stagingPath := filepath.Join(inst.StagingDir, filePath)
				logging.GlobalLogger.Info(fmt.Sprintf("File complete, verifying: %s", filePath))

				f, err := os.Open(stagingPath)
				if err != nil {
					logging.GlobalLogger.Error(fmt.Sprintf("Failed to open completed file %s: %v - re-enqueueing all chunks for this file", stagingPath, err))

					delete(fileAssembledChunks, filePath)
					delete(fileExpectedChunks, filePath)

					if removeErr := os.Remove(stagingPath); removeErr != nil && !os.IsNotExist(removeErr) {
						logging.GlobalLogger.Warn(fmt.Sprintf("Failed to remove corrupted staging file %s: %v", stagingPath, removeErr))
					}

					for _, chunkID := range fileMeta.Chunks {
						chunkMeta := inst.ChunkMap[chunkID]

						var offset uint64
						var found bool
						for _, d := range chunkMeta.Destinations {
							if d.File == fileMeta {
								offset = d.Offset
								found = true
								break
							}
						}
						if !found {
							logging.GlobalLogger.Fatal(fmt.Sprintf("Offset not found for file %s in chunk %s", fileMeta.FilePath, chunkID))
							return
						}

						// Create new ChunkMetaData for re-enqueueing (only for this file)
						cm_new := &ChunkMetaData{
							ChunkID:          chunkMeta.ChunkID,
							URL:              chunkMeta.URL,
							MD5:              chunkMeta.MD5,
							CompressedSize:   chunkMeta.CompressedSize,
							UncompressedSize: chunkMeta.UncompressedSize,
							IsCompressed:     chunkMeta.IsCompressed,
							Destinations: []ChunkDestination{
								{File: fileMeta, Offset: offset},
							},
						}

						utils.NonBlockingEnqueue(inst.InputQueue, ChunksInput{Metadata: cm_new})

						inst.Progress.mu.Lock()
						inst.Progress.TotalBytes += int64(chunkMeta.CompressedSize)
						inst.Progress.mu.Unlock()
					}
					continue
				}

				inst.Verifier2.EnqueueVerification(filePath, f, fileMeta.MD5, fileMeta)

				delete(fileAssembledChunks, filePath)
				delete(fileExpectedChunks, filePath)
			}
		}
		logging.GlobalLogger.Info("Assembler output closed, stopping File Verifier")
		inst.Verifier2.Stop()
	}()
}

func (inst *Installer) MoveFiles() {
	logging.GlobalLogger.Info("Starting file move to game directory")

	inst.wg.Add(1)
	go func() {
		defer inst.wg.Done()
		for verifyOutput := range inst.Verifier2.GetOutputChannel() {
			fm := verifyOutput.Payload.(*FileMetaData)
			stagingPath := filepath.Join(inst.StagingDir, fm.FilePath)
			finalPath := filepath.Join(inst.GameDir, fm.FilePath)

			if !verifyOutput.Suceeded {
				logging.GlobalLogger.Error(fmt.Sprintf("File verification failed: %s - re-enqueueing all chunks", fm.FilePath))

				if removeErr := os.Remove(stagingPath); removeErr != nil && !os.IsNotExist(removeErr) {
					logging.GlobalLogger.Warn(fmt.Sprintf("Failed to remove corrupted staging file %s: %v", stagingPath, removeErr))
				}

				for _, chunkID := range fm.Chunks {
					cm := inst.ChunkMap[chunkID]

					var offset uint64
					var offsetFound bool
					for _, dest := range cm.Destinations {
						if dest.File == fm {
							offset = dest.Offset
							offsetFound = true
							break
						}
					}
					if !offsetFound {
						logging.GlobalLogger.Fatal(fmt.Sprintf("Offset not found for file %s in chunk %s", fm.FilePath, chunkID))
					}

					new_cm := &ChunkMetaData{
						ChunkID:          cm.ChunkID,
						URL:              cm.URL,
						MD5:              cm.MD5,
						CompressedSize:   cm.CompressedSize,
						UncompressedSize: cm.UncompressedSize,
						IsCompressed:     cm.IsCompressed,
						Destinations: []ChunkDestination{
							{File: fm, Offset: offset},
						},
					}

					utils.NonBlockingEnqueue(inst.InputQueue, ChunksInput{Metadata: new_cm})

					inst.Progress.mu.Lock()
					inst.Progress.TotalBytes += int64(cm.CompressedSize)
					inst.Progress.mu.Unlock()
				}
				continue
			}
			logging.GlobalLogger.Info(fmt.Sprintf("File verified successfully: %s", fm.FilePath))

			finalDir := filepath.Dir(finalPath)
			if err := os.MkdirAll(finalDir, 0o755); err != nil {
				logging.GlobalLogger.Fatal(fmt.Sprintf("Failed to create directory for final file location: %s : %v", finalDir, err))
				return
			}

			err := os.Rename(stagingPath, finalPath)
			if err != nil {
				logging.GlobalLogger.Fatal(fmt.Sprintf("Failed to move file from staging to final location: %s -> %s : %v", stagingPath, finalPath, err))
				return
			}

			inst.Progress.mu.Lock()
			inst.Progress.VerifiedFiles++
			verifiedFiles := inst.Progress.VerifiedFiles
			totalFiles := inst.Progress.TotalFiles
			inst.Progress.mu.Unlock()

			if verifiedFiles >= totalFiles {
				logging.GlobalLogger.Info("All files verified and moved, closing InputQueue to shut down pipeline")
				close(inst.InputQueue)
			}
		}
		logging.GlobalLogger.Info("File Verifier output closed, file move complete")
	}()
}
