package operations

import (
	"SophonClientv2/internal/logging"
	"SophonClientv2/pkg/assembler"
	"SophonClientv2/pkg/verifier"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

func (inst *Installer) DownloadChunks() {
	if inst.Downloader == nil {
		logging.GlobalLogger.Fatal("Downloader not initialized. Something is wrong with the code.")
		return
	}
	logging.GlobalLogger.Info("Starting chunk download")

	orderedChunks := inst.EnumerateChunksWithFileOrder()
	if len(orderedChunks) != len(inst.ChunkMap) {
		logging.GlobalLogger.Fatal("Chunk enumeration mismatch. Something is wrong with the code.")
		return
	}
	for _, cm := range orderedChunks {
		inst.Downloader.EnqueueDownload(cm.URL, cm)
	}

	logging.GlobalLogger.Info("All chunks enqueued for download")
	return
}

func (inst *Installer) DecompressChunks() {
	if inst.Decompressor == nil {
		logging.GlobalLogger.Fatal("Decompressor not initialized. Something is wrong with the code.")
		return
	}
	if inst.Downloader == nil {
		logging.GlobalLogger.Fatal("Downloader not initialized. Something is wrong with the code.")
		return
	}

	logging.GlobalLogger.Info("Starting chunk decompression")

	// Consume from downloader output and feed to decompressor
	jobs := inst.Progress.TotalChunks
	processedChunks := 0

	for processedChunks < jobs {
		downloadOutput := <-inst.Downloader.GetOutputChannel()

		if !downloadOutput.Suceeded {
			cm := downloadOutput.Payload.(*ChunkMetaData)
			logging.GlobalLogger.Warn(fmt.Sprintf("Download failed for chunk %s, re-enqueueing", cm.ChunkID))
			inst.Downloader.EnqueueDownload(cm.URL, cm)
			continue
		}

		cm := downloadOutput.Payload.(*ChunkMetaData)
		processedChunks++

		if cm.IsCompressed {
			inst.Decompressor.EnqueueDecompression(downloadOutput.Content, cm)
		} else {
			// TODO: Handle uncompressed chunks (pass through)
			logging.GlobalLogger.Fatal("Uncompressed chunks are not yet supported")
			return
		}
	}

	logging.GlobalLogger.Info("All chunks enqueued for decompression")
	return
}

func (inst *Installer) VerifyChunks() {
	if inst.Decompressor == nil {
		logging.GlobalLogger.Fatal("Decompressor not initialized. Something is wrong with the code.")
		return
	}
	if inst.Downloader == nil {
		logging.GlobalLogger.Fatal("Downloader not initialized. Something is wrong with the code.")
		return
	}
	if inst.Verifier == nil {
		logging.GlobalLogger.Fatal("Verifier not initialized. Something is wrong with the code.")
		return
	}

	logging.GlobalLogger.Info("Starting chunk verification")

	jobs := inst.Progress.TotalChunks
	processedChunks := 0

	for processedChunks < jobs {
		decompressOutput := <-inst.Decompressor.GetOutputChannel()

		if !decompressOutput.Suceeded {
			cm := decompressOutput.Payload.(*ChunkMetaData)
			logging.GlobalLogger.Warn(fmt.Sprintf("Decompression failed for chunk %s, re-enqueueing", cm.ChunkID))
			inst.Downloader.EnqueueDownload(cm.URL, cm)
			continue
		}

		cm := decompressOutput.Payload.(*ChunkMetaData)
		processedChunks++

		inst.Verifier.EnqueueVerification(cm.ChunkID, decompressOutput.Content, cm.MD5, cm)
	}

	logging.GlobalLogger.Info("All chunks enqueued for verification")
}

func (inst *Installer) AssembleChunks() {
	if inst.Verifier == nil {
		logging.GlobalLogger.Fatal("Verifier not initialized. Something is wrong with the code.")
		return
	}
	if inst.Downloader == nil {
		logging.GlobalLogger.Fatal("Downloader not initialized. Something is wrong with the code.")
		return
	}

	logging.GlobalLogger.Info("Starting chunk assembly")

	inst.Assembler = assembler.NewAssembler(inst.StagingDir, inst.Progress.TotalChunks)
	jobs := inst.Progress.TotalChunks
	processedChunks := 0

	for processedChunks < jobs {
		verifierOutput := <-inst.Verifier.GetOutputChannel()

		if !verifierOutput.Suceeded {
			cm := verifierOutput.Payload.(*ChunkMetaData)
			logging.GlobalLogger.Warn(fmt.Sprintf("Chunk verification failed for %s, re-enqueueing", cm.ChunkID))
			inst.Downloader.EnqueueDownload(cm.URL, cm)
			continue
		}

		cm := verifierOutput.Payload.(*ChunkMetaData)
		processedChunks++

		for _, dest := range cm.Destinations {
			inst.Assembler.EnqueueWrite(dest.File.FilePath, dest.Offset, cm.ChunkID, verifierOutput.Content, cm)
			break
		}
	}

	logging.GlobalLogger.Info("All chunks enqueued for assembly")
}

func (inst *Installer) VerifyFiles() {
	if inst.Assembler == nil {
		logging.GlobalLogger.Fatal("Assembler not initialized. Something is wrong with the code.")
		return
	}
	if inst.Downloader == nil {
		logging.GlobalLogger.Fatal("Downloader not initialized. Something is wrong with the code.")
		return
	}

	logging.GlobalLogger.Info("Starting file verification goroutine")

	fileChunkCount := make(map[string]int)
	fileChunkCountMu := sync.Mutex{}

	fileVerifier := verifier.NewVerifier(len(inst.FileMap) + 10)

	jobs := inst.Progress.TotalChunks
	processedChunks := 0

	go func() {
		for i := 0; i < jobs; i++ {
			assemblerOutput := <-inst.Assembler.GetOutputChannel()

			if !assemblerOutput.Succeeded {
				cm := assemblerOutput.Payload.(*ChunkMetaData)
				logging.GlobalLogger.Warn(fmt.Sprintf("Chunk assembly failed for %s, re-enqueueing", cm.ChunkID))
				inst.Downloader.EnqueueDownload(cm.URL, cm)
				continue
			}

			cm := assemblerOutput.Payload.(*ChunkMetaData)
			processedChunks++

			fileChunkCountMu.Lock()
			for _, dest := range cm.Destinations {
				filePath := dest.File.FilePath
				fileChunkCount[filePath]++

				if fileChunkCount[filePath] == len(dest.File.Chunks) {
					logging.GlobalLogger.Info(fmt.Sprintf("File complete, verifying: %s", filePath))

					stagingPath := filepath.Join(inst.StagingDir, filePath)
					f, err := os.Open(stagingPath)
					if err != nil {
						logging.GlobalLogger.Error(fmt.Sprintf("Failed to open completed file %s: %v", stagingPath, err))
						for _, chunkID := range dest.File.Chunks {
							if chunkMeta, ok := inst.ChunkMap[chunkID]; ok {
								inst.Downloader.EnqueueDownload(chunkMeta.URL, chunkMeta)
							}
						}
						fileChunkCount[filePath] = 0
						continue
					}

					fileVerifier.EnqueueVerification(filePath, f, dest.File.MD5, dest.File)
				}
			}
			fileChunkCountMu.Unlock()

			logging.GlobalLogger.Debug(fmt.Sprintf("Processed chunk %s (%d/%d)", cm.ChunkID, processedChunks, jobs))
		}

		logging.GlobalLogger.Debug("All chunks processed, waiting for file verifications to complete")
	}()

	expectedFiles := len(inst.FileMap)
	verifiedFiles := 0

	for verifiedFiles < expectedFiles {
		verifyResult := <-fileVerifier.GetOutputChannel()
		fm := verifyResult.Payload.(*FileMetaData)

		if !verifyResult.Suceeded {
			logging.GlobalLogger.Error(fmt.Sprintf("File verification failed: %s - re-enqueueing all chunks", fm.FilePath))
			fileChunkCountMu.Lock()
			for _, chunkID := range fm.Chunks {
				if chunkMeta, ok := inst.ChunkMap[chunkID]; ok {
					inst.Downloader.EnqueueDownload(chunkMeta.URL, chunkMeta)
				}
			}
			fileChunkCount[fm.FilePath] = 0
			fileChunkCountMu.Unlock()
		} else {
			logging.GlobalLogger.Info(fmt.Sprintf("File verified successfully: %s", fm.FilePath))

			stagingPath := filepath.Join(inst.StagingDir, fm.FilePath)
			finalPath := filepath.Join(inst.GameDir, fm.FilePath)

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

			verifiedFiles++
		}
	}

	fileVerifier.Stop()
	inst.Assembler.Stop()
	logging.GlobalLogger.Info("All files assembled and verified")
}
