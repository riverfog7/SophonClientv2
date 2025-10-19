package operations

import (
	"SophonClientv2/internal/logging"
	"SophonClientv2/internal/models"
	"SophonClientv2/pkg/verifier"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func (inst *Installer) ParseManifest(mani *models.Manifest, chunkDownload models.SophonChunkDownloadInfo) error {
	logging.GlobalLogger.Debug("Resetting installer state before parsing manifest")
	inst.ChunkMap = make(map[string]*ChunkMetaData)
	inst.FileMap = make(map[string]*FileMetaData)
	inst.Progress = InstallProgress{}

	for _, fi := range mani.GetFiles() {
		filePath := fi.GetFilename()
		isFolder := fi.GetFlags() == 64
		fm := &FileMetaData{
			FilePath: filePath,
			Size:     fi.GetSize(),
			MD5:      fi.GetMd5(),
			Chunks:   make([]string, len(fi.GetChunks())),
			IsFolder: isFolder,
		}

		for i, ci := range fi.GetChunks() {
			chunkID := ci.GetChunkId()
			fm.Chunks[i] = chunkID

			if _, ok := inst.ChunkMap[chunkID]; !ok {
				url := chunkDownload.UrlPrefix + "/" + chunkID
				if chunkDownload.UrlSuffix != "" {
					url += "/" + chunkDownload.UrlSuffix
				}

				inst.ChunkMap[chunkID] = &ChunkMetaData{
					ChunkID:          chunkID,
					URL:              url,
					MD5:              ci.GetMd5(),
					CompressedSize:   ci.GetCompressedSize(),
					UncompressedSize: ci.GetUncompressedSize(),
					Destinations:     []ChunkDestination{{File: fm, Offset: ci.GetOffset()}},
					IsCompressed:     chunkDownload.Compression != 0,
				}
			} else {
				inst.ChunkMap[chunkID].Destinations = append(
					inst.ChunkMap[chunkID].Destinations,
					ChunkDestination{File: fm, Offset: ci.GetOffset()},
				)
			}
		}
		inst.FileMap[filePath] = fm
	}
	inst.Progress.TotalChunks = len(inst.ChunkMap)
	inst.ComputeTotalBytes()
	logging.GlobalLogger.Info(fmt.Sprintf("Parsed manifest: %d chunks for %d files, total %d bytes", inst.Progress.TotalChunks, len(inst.FileMap), inst.Progress.TotalBytes))

	var totalChunksInManifest int
	for _, f := range mani.GetFiles() {
		totalChunksInManifest += len(f.GetChunks())
	}
	logging.GlobalLogger.Debug(fmt.Sprintf("Total chunks in manifest before deduplication: %d", totalChunksInManifest))
	return nil
}

func (inst *Installer) ComputeTotalBytes() {
	logging.GlobalLogger.Debug("Recomputing total bytes from ChunkMap")
	var total int64
	for _, chunk := range inst.ChunkMap {
		// Download size is compressed size
		total += int64(chunk.CompressedSize)
	}
	inst.Progress.mu.Lock()
	inst.Progress.TotalBytes = total
	inst.Progress.mu.Unlock()
	logging.GlobalLogger.Debug(fmt.Sprintf("Total bytes to download: %d", total))
}

func (inst *Installer) Prepare() error {
	// Clear staging directory (remove previous probably failed downloads)
	logging.GlobalLogger.Info("Clearing staging directory")
	os.RemoveAll(inst.StagingDir) // Ignore error (Somehow fails on my machine)

	if err := os.MkdirAll(inst.StagingDir, 0o755); err != nil {
		logging.GlobalLogger.Error(fmt.Sprintf("Error creating staging dir: %v", err))
		return fmt.Errorf("creating staging dir: %w", err)
	}

	// Set up verifier and enqueue existing files
	ver := verifier.NewVerifier(len(inst.FileMap) + 10)
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

	inst.Progress.TotalChunks = len(inst.ChunkMap)
	inst.ComputeTotalBytes()
	logging.GlobalLogger.Info(fmt.Sprintf("Prepare complete, %d chunks, %d files, total %d bytes remaining", inst.Progress.TotalChunks, len(inst.FileMap), inst.Progress.TotalBytes))
	return nil
}

func (inst *Installer) EnumerateChunksWithFileOrder() []*ChunkMetaData {
	positiveSubstrs := []string{"globalgame", "pkg_version", "data.unity3d", "exe"}
	negativeSubstrs := []string{"/"}

	calculatePriority := func(filePath string) int {
		priority := 0
		for _, substr := range positiveSubstrs {
			if contains(filePath, substr) {
				priority += 10000
			}
		}
		for _, substr := range negativeSubstrs {
			if contains(filePath, substr) {
				priority -= 100
			}
		}
		return priority
	}

	type filePriority struct {
		file     *FileMetaData
		priority int
	}

	fileList := make([]filePriority, 0, len(inst.FileMap))
	for _, fm := range inst.FileMap {
		fileList = append(fileList, filePriority{
			file:     fm,
			priority: calculatePriority(fm.FilePath),
		})
	}

	sort.Slice(fileList, func(i, j int) bool {
		if fileList[i].priority != fileList[j].priority {
			return fileList[i].priority > fileList[j].priority
		}
		return fileList[i].file.FilePath < fileList[j].file.FilePath
	})

	addedChunks := make(map[string]bool)
	chunkList := make([]*ChunkMetaData, 0, len(inst.ChunkMap))

	for _, fp := range fileList {
		for _, chunkID := range fp.file.Chunks {
			if !addedChunks[chunkID] {
				if cm, ok := inst.ChunkMap[chunkID]; ok {
					chunkList = append(chunkList, cm)
					addedChunks[chunkID] = true
				}
			}
		}
	}

	logging.GlobalLogger.Info(fmt.Sprintf("Enumerated %d chunks in priority order from %d files", len(chunkList), len(fileList)))
	return chunkList
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) &&
		(str == substr ||
			len(str) > len(substr) &&
				(hasSubstring(str, substr)))
}

func hasSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
