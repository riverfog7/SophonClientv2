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
		inst.setTerminalError(fmt.Errorf("chunk enumeration mismatch: ordered=%d mapped=%d", len(orderedChunks), len(inst.ChunkMap)))
		return
	}

	if len(orderedChunks) == 0 {
		logging.GlobalLogger.Info("No chunks to download, nothing to enqueue")
		inst.closeInputQueue()
		return
	}

	inst.wg.Add(1)
	go func() {
		defer inst.wg.Done()
		for _, cm := range orderedChunks {
			if inst.hasTerminalError() {
				break
			}
			if !inst.enqueueChunkInput(ChunksInput{Metadata: cm}) {
				inst.setTerminalError(fmt.Errorf("failed to enqueue initial chunk %s", cm.ChunkID))
				break
			}
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
			if inst.hasTerminalError() {
				continue
			}
			if input.Metadata == nil {
				inst.setTerminalError(fmt.Errorf("received nil chunk metadata in download stage"))
				continue
			}
			if input.Metadata.URL == "" {
				inst.setTerminalError(fmt.Errorf("empty chunk URL for chunk %s", input.Metadata.ChunkID))
				continue
			}
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
			cm, ok := inst.chunkPayload(downloadOutput.Payload, "download-output")
			if !ok {
				utils.CloseQuietly(downloadOutput.Content)
				continue
			}
			if inst.hasTerminalError() {
				utils.CloseQuietly(downloadOutput.Content)
				continue
			}

			if !downloadOutput.Suceeded {
				logging.GlobalLogger.Warn(fmt.Sprintf("Download failed for chunk %s, re-enqueueing", cm.ChunkID))
				utils.CloseQuietly(downloadOutput.Content)
				inst.tryRequeueChunk(cm, "download")
				continue
			}

			if cm.IsCompressed {
				inst.Decompressor.EnqueueDecompression(downloadOutput.Content, cm)

				inst.Progress.IncrementDownloadedChunks()
				inst.Progress.IncrementDownloadedBytes(int64(cm.CompressedSize))
			} else {
				utils.CloseQuietly(downloadOutput.Content)
				inst.setTerminalError(fmt.Errorf("unsupported uncompressed chunk: %s", cm.ChunkID))
				continue
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
			cm, ok := inst.chunkPayload(decompressOutput.Payload, "decompress-output")
			if !ok {
				utils.CloseQuietly(decompressOutput.Content)
				continue
			}
			if inst.hasTerminalError() {
				utils.CloseQuietly(decompressOutput.Content)
				continue
			}

			if !decompressOutput.Suceeded {
				logging.GlobalLogger.Warn(fmt.Sprintf("Decompression failed for chunk %s, re-enqueueing", cm.ChunkID))
				utils.CloseQuietly(decompressOutput.Content)
				if inst.tryRequeueChunk(cm, "decompress") {
					inst.Progress.IncrementTotalBytes(int64(cm.CompressedSize))
				}

				continue
			}

			inst.Verifier.EnqueueVerification(cm.ChunkID, decompressOutput.Content, cm.MD5, cm)
			inst.Progress.IncrementDecompressedChunks()
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
			cm, ok := inst.chunkPayload(verifyOutput.Payload, "chunk-verify-output")
			if !ok {
				utils.CloseQuietly(verifyOutput.Content)
				continue
			}
			if inst.hasTerminalError() {
				utils.CloseQuietly(verifyOutput.Content)
				continue
			}

			if !verifyOutput.Suceeded {
				logging.GlobalLogger.Warn(fmt.Sprintf("Verification failed for chunk %s, re-enqueueing", cm.ChunkID))
				utils.CloseQuietly(verifyOutput.Content)
				if inst.tryRequeueChunk(cm, "chunk-verify") {
					inst.Progress.IncrementTotalBytes(int64(cm.CompressedSize))
				}

				continue
			}
			inst.Progress.IncrementVerifiedChunks()

			// This is here because one chunk can be used for multiple files.
			// And readcloser can only be read once.
			// Read content into memory once for reuse across multiple destinations
			contentBytes, err := io.ReadAll(verifyOutput.Content)
			utils.CloseQuietly(verifyOutput.Content)

			if err != nil {
				logging.GlobalLogger.Error(fmt.Sprintf("Failed to read verified content for chunk %s: %v, re-enqueueing", cm.ChunkID, err))
				contentBytes = nil // Content is already closed by readAll or CloseQuietly
				if inst.tryRequeueChunk(cm, "chunk-read") {
					inst.Progress.IncrementTotalBytes(int64(cm.CompressedSize))
				}
				continue
			}

			// Create a new reader for each destination
			validDestinations := true
			for _, dest := range cm.Destinations {
				if dest.File == nil || dest.File.FilePath == "" {
					inst.setTerminalError(fmt.Errorf("invalid destination for chunk %s", cm.ChunkID))
					validDestinations = false
					break
				}
			}
			if !validDestinations {
				continue
			}

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
			cm, ok := inst.chunkPayload(assemblerOutput.Payload, "assembler-output")
			if !ok {
				continue
			}
			filePath := assemblerOutput.FilePath
			if inst.hasTerminalError() {
				continue
			}
			if filePath == "" {
				inst.setTerminalError(fmt.Errorf("empty file path in assembler output for chunk %s", cm.ChunkID))
				continue
			}

			if !assemblerOutput.Succeeded {
				logging.GlobalLogger.Warn(fmt.Sprintf("Assembly failed for chunk %s, re-enqueueing", cm.ChunkID))
				if inst.tryRequeueChunk(cm, "assemble") {
					inst.Progress.IncrementTotalBytes(int64(cm.CompressedSize))
				}
				continue
			}
			inst.Progress.IncrementAssembledChunks()

			if fileAssembledChunks[filePath] == nil {
				fileAssembledChunks[filePath] = make(map[string]bool)
			}

			// Find the offset for this specific file to create a unique key
			var offset uint64
			var fileMeta *FileMetaData
			for _, dest := range cm.Destinations {
				if dest.File == nil {
					continue
				}
				if dest.File.FilePath == filePath {
					fileMeta = dest.File
					offset = dest.Offset
					break
				}
			}
			if fileMeta == nil {
				inst.setTerminalError(fmt.Errorf("file metadata not found for assembled file: %s", filePath))
				continue
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
				if !inst.tryMarkFileVerifyInFlight(filePath) {
					delete(fileAssembledChunks, filePath)
					delete(fileExpectedChunks, filePath)
					continue
				}

				stagingPath := filepath.Join(inst.StagingDir, filePath)
				logging.GlobalLogger.Info(fmt.Sprintf("File complete, verifying: %s", filePath))

				f, err := os.Open(stagingPath)
				if err != nil {
					logging.GlobalLogger.Error(fmt.Sprintf("Failed to open completed file %s: %v - re-enqueueing all chunks for this file", stagingPath, err))
					inst.clearFileVerifyInFlight(filePath)

					delete(fileAssembledChunks, filePath)
					delete(fileExpectedChunks, filePath)

					if removeErr := os.Remove(stagingPath); removeErr != nil && !os.IsNotExist(removeErr) {
						logging.GlobalLogger.Warn(fmt.Sprintf("Failed to remove corrupted staging file %s: %v", stagingPath, removeErr))
					}

					if !inst.tryRequeueFile(fileMeta.FilePath, "verify-file-open") {
						continue
					}

					for _, chunkID := range fileMeta.Chunks {
						chunkMeta, ok := inst.ChunkMap[chunkID]
						if !ok {
							inst.setTerminalError(fmt.Errorf("chunk metadata not found for %s (%s)", fileMeta.FilePath, chunkID))
							break
						}

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
							inst.setTerminalError(fmt.Errorf("offset not found for file %s in chunk %s", fileMeta.FilePath, chunkID))
							break
						}

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

						if inst.tryRequeueChunk(cm_new, "verify-file-open") {
							inst.Progress.IncrementTotalBytes(int64(chunkMeta.CompressedSize))
						}
						if inst.hasTerminalError() {
							break
						}
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
			if inst.hasTerminalError() {
				continue
			}

			fm, ok := inst.filePayload(verifyOutput.Payload, "file-verify-output")
			if !ok {
				continue
			}
			if fm.FilePath == "" {
				inst.setTerminalError(fmt.Errorf("empty file path in file verifier output"))
				continue
			}
			if inst.isFileCompleted(fm.FilePath) {
				inst.clearFileVerifyInFlight(fm.FilePath)
				continue
			}

			stagingPath := filepath.Join(inst.StagingDir, fm.FilePath)
			finalPath := filepath.Join(inst.GameDir, fm.FilePath)

			if !verifyOutput.Suceeded {
				logging.GlobalLogger.Error(fmt.Sprintf("File verification failed: %s - re-enqueueing all chunks", fm.FilePath))
				inst.clearFileVerifyInFlight(fm.FilePath)
				if !inst.tryRequeueFile(fm.FilePath, "file-verify") {
					continue
				}

				if removeErr := os.Remove(stagingPath); removeErr != nil && !os.IsNotExist(removeErr) {
					logging.GlobalLogger.Warn(fmt.Sprintf("Failed to remove corrupted staging file %s: %v", stagingPath, removeErr))
				}

				for _, chunkID := range fm.Chunks {
					cm, ok := inst.ChunkMap[chunkID]
					if !ok {
						inst.setTerminalError(fmt.Errorf("chunk metadata not found for %s (%s)", fm.FilePath, chunkID))
						break
					}

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
						inst.setTerminalError(fmt.Errorf("offset not found for file %s in chunk %s", fm.FilePath, chunkID))
						break
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

					if inst.tryRequeueChunk(new_cm, "file-verify") {
						inst.Progress.IncrementTotalBytes(int64(cm.CompressedSize))
					}
					if inst.hasTerminalError() {
						break
					}
				}
				continue
			}
			logging.GlobalLogger.Info(fmt.Sprintf("File verified successfully: %s", fm.FilePath))

			finalDir := filepath.Dir(finalPath)
			if err := os.MkdirAll(finalDir, 0o755); err != nil {
				inst.setTerminalError(fmt.Errorf("failed to create final directory %s: %w", finalDir, err))
				continue
			}

			err := os.Rename(stagingPath, finalPath)
			if err != nil {
				inst.setTerminalError(fmt.Errorf("failed to move file %s -> %s: %w", stagingPath, finalPath, err))
				continue
			}

			newlyCompleted, completedCount := inst.markFileCompleted(fm.FilePath)
			if !newlyCompleted {
				continue
			}

			inst.Progress.mu.Lock()
			inst.Progress.VerifiedFiles = completedCount
			verifiedFiles := completedCount
			totalFiles := inst.Progress.TotalFiles
			inst.Progress.mu.Unlock()

			if verifiedFiles >= totalFiles {
				logging.GlobalLogger.Info("All files verified and moved, closing InputQueue to shut down pipeline")
				inst.closeInputQueue()
			}
		}
		logging.GlobalLogger.Info("File Verifier output closed, file move complete")
	}()
}
