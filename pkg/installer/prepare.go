package installer

import (
	"SophonClientv2/internal/logging"
	"SophonClientv2/pkg/verifier"
	"fmt"
	"os"
	"path/filepath"
)

func (inst *Installer) Prepare() error {
	// Clear staging directory (remove previous probably failed downloads)
	logging.GlobalLogger.Info("Clearing staging directory")
	os.RemoveAll(inst.StagingDir) // Ignore error (Somehow fails on my machine)

	if err := os.MkdirAll(inst.StagingDir, 0o755); err != nil {
		logging.GlobalLogger.Error(fmt.Sprintf("Error creating staging dir: %v", err))
		return fmt.Errorf("creating staging dir: %w", err)
	}

	// Set up verifier and enqueue existing files
	// Queue size should be enough to hold all files (No subscriber for output yet)
	ver := verifier.NewVerifier(len(inst.FileMap)+10, false)
	jobs := 0
	for filePath, fm := range inst.FileMap {
		absPath := filepath.Join(inst.GameDir, filePath)
		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				logging.GlobalLogger.Debug(fmt.Sprintf("File not present, will download: %s", absPath))
				continue
			}
			logging.GlobalLogger.Fatal(fmt.Sprintf("Error stating file %s: %v", absPath, err))
			return fmt.Errorf("stat existing file %s: %w", absPath, err)
		}
		if info.IsDir() {
			logging.GlobalLogger.Debug(fmt.Sprintf("Skipping directory entry: %s", absPath))
			continue
		}

		// MD5 hashcheck (submits to verifier)
		f, err := os.Open(absPath)
		if err != nil {
			logging.GlobalLogger.Fatal(fmt.Sprintf("Error opening existing file %s: %v", absPath, err))
			return fmt.Errorf("opening existing file %s: %w", absPath, err)
		}
		ver.EnqueueVerification(f.Name(), f, fm.MD5, fm)
		jobs++
	}

	// Collect verifier results
	for i := 0; i < jobs; i++ {
		out := <-ver.GetOutputChannel()
		fmOut := out.Payload.(*FileMetaData)
		absPath := filepath.Join(inst.GameDir, fmOut.FilePath)

		if out.Suceeded {
			logging.GlobalLogger.Debug(fmt.Sprintf("Existing file verified, skipping download: %s", absPath))

			for _, chunkID := range fmOut.Chunks {
				if cm, ok := inst.ChunkMap[chunkID]; ok {
					newD := make([]ChunkDestination, 0, len(cm.Destinations))
					for _, dest := range cm.Destinations {
						if dest.File != fmOut {
							newD = append(newD, dest)
						}
					}
					if len(newD) == 0 {
						delete(inst.ChunkMap, chunkID)
					} else {
						cm.Destinations = newD
					}
				}
			}
			delete(inst.FileMap, fmOut.FilePath)
		} else {
			logging.GlobalLogger.Warn(fmt.Sprintf("File failed verification, deleting: %s", absPath))
			if err := os.Remove(absPath); err != nil {
				logging.GlobalLogger.Fatal(fmt.Sprintf("Error deleting file %s: %v", absPath, err))
				return fmt.Errorf("deleting invalid file %s: %w", absPath, err)
			}
		}
	}
	ver.Stop()
	inst.Progress.mu.Lock()
	inst.Progress.TotalChunks = len(inst.ChunkMap)
	inst.Progress.TotalFiles = len(inst.FileMap)
	inst.Progress.mu.Unlock()
	inst.ComputeTotalBytes()
	logging.GlobalLogger.Info(fmt.Sprintf("Prepare complete, %d chunks, %d files, total %d bytes remaining", inst.Progress.TotalChunks, len(inst.FileMap), inst.Progress.TotalBytes))
	return nil
}
