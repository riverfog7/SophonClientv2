package installer

import (
	"SophonClientv2/internal/logging"
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

	go func() {
		for _, cm := range orderedChunks {
			inst.InputQueue <- ChunksInput{
				Metadata: cm,
			}
		}
	}()
}

func (inst *Installer) DownloadChunks() {
	logging.GlobalLogger.Info("Starting chunk download")

	go func() {
		for input := range inst.InputQueue {
			inst.Downloader.EnqueueDownload(input.Metadata.URL, input.Metadata)
		}
	}()
}

func (inst *Installer) DecompressChunks() {
	logging.GlobalLogger.Info("Starting chunk decompression")

	go func() {
		for downloadOutput := range inst.Downloader.GetOutputChannel() {
			cm := downloadOutput.Payload.(*ChunkMetaData)

			if !downloadOutput.Suceeded {
				logging.GlobalLogger.Warn(fmt.Sprintf("Download failed for chunk %s, re-enqueueing", cm.ChunkID))

				inst.InputQueue <- ChunksInput{
					Metadata: cm,
				}
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
	}()
}

func (inst *Installer) VerifyChunks() {
	logging.GlobalLogger.Info("Starting chunk verification")

	go func() {
		for decompressOutput := range inst.Decompressor.GetOutputChannel() {
			cm := decompressOutput.Payload.(*ChunkMetaData)

			if !decompressOutput.Suceeded {
				logging.GlobalLogger.Warn(fmt.Sprintf("Decompression failed for chunk %s, re-enqueueing", cm.ChunkID))

				inst.InputQueue <- ChunksInput{
					Metadata: cm,
				}

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
	}()
}

func (inst *Installer) AssembleChunks() {
	logging.GlobalLogger.Info("Starting chunk assembly")

	go func() {
		for verifyOutput := range inst.Verifier.GetOutputChannel() {
			cm := verifyOutput.Payload.(*ChunkMetaData)

			if !verifyOutput.Suceeded {
				logging.GlobalLogger.Warn(fmt.Sprintf("Verification failed for chunk %s, re-enqueueing", cm.ChunkID))

				inst.InputQueue <- ChunksInput{
					Metadata: cm,
				}

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
				inst.InputQueue <- ChunksInput{
					Metadata: cm,
				}
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
	}()
}

func (inst *Installer) VerifyFiles() {
	logging.GlobalLogger.Info("Starting file verification")

	go func() {
		fileChunkCount := make(map[string]int)

		for assemblerOutput := range inst.Assembler.GetOutputChannel() {
			cm := assemblerOutput.Payload.(*ChunkMetaData)

			if !assemblerOutput.Succeeded {
				logging.GlobalLogger.Warn(fmt.Sprintf("Assembly failed for chunk %s, re-enqueueing", cm.ChunkID))

				inst.InputQueue <- ChunksInput{
					Metadata: cm,
				}

				// Adjust downloaded bytes since we are re-enqueueing
				inst.Progress.mu.Lock()
				inst.Progress.TotalBytes += int64(cm.CompressedSize)
				inst.Progress.mu.Unlock()
				continue
			}

			inst.Progress.mu.Lock()
			inst.Progress.AssembledChunks++
			inst.Progress.mu.Unlock()

			for _, dest := range cm.Destinations {
				fileChunkCount[dest.File.FilePath]++

				// Check if all chunks for the file are assembled
				if fileChunkCount[dest.File.FilePath] == len(dest.File.Chunks) {
					stagingPath := filepath.Join(inst.StagingDir, dest.File.FilePath)
					logging.GlobalLogger.Info(fmt.Sprintf("File complete, verifying: %s", dest.File.FilePath))

					f, err := os.Open(stagingPath)
					if err != nil {
						logging.GlobalLogger.Error(fmt.Sprintf("Failed to open completed file %s: %v", stagingPath, err))

						fileChunkCount[dest.File.FilePath] = 0
						inst.Progress.mu.Lock()
						inst.Progress.TotalBytes += int64(cm.CompressedSize)
						inst.Progress.mu.Unlock()

						// Create new ChunkMetaData for re-enqueueing (only for this file)
						cm_new := &ChunkMetaData{
							ChunkID:          cm.ChunkID,
							URL:              cm.URL,
							MD5:              cm.MD5,
							CompressedSize:   cm.CompressedSize,
							UncompressedSize: cm.UncompressedSize,
							IsCompressed:     cm.IsCompressed,
							Destinations: []ChunkDestination{
								{File: dest.File, Offset: dest.Offset},
							},
						}
						inst.InputQueue <- ChunksInput{
							Metadata: cm_new,
						}
						continue
					}

					inst.Verifier2.EnqueueVerification(dest.File.FilePath, f, dest.File.MD5, dest.File)
				}
			}
		}
	}()
}

func (inst *Installer) MoveFiles() {
	logging.GlobalLogger.Info("Starting file move to game directory")

	go func() {
		for verifyOutput := range inst.Verifier2.GetOutputChannel() {
			fm := verifyOutput.Payload.(*FileMetaData)
			stagingPath := filepath.Join(inst.StagingDir, fm.FilePath)
			finalPath := filepath.Join(inst.GameDir, fm.FilePath)

			if !verifyOutput.Suceeded {
				logging.GlobalLogger.Error(fmt.Sprintf("File verification failed: %s - re-enqueueing all chunks", fm.FilePath))

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

					inst.InputQueue <- ChunksInput{
						Metadata: new_cm,
					}

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
			inst.Progress.mu.Unlock()
		}
	}()
}
