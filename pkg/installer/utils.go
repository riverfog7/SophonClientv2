package installer

import (
	"SophonClientv2/internal/logging"
	"SophonClientv2/internal/models"
	"fmt"
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
	inst.Progress.mu.Lock()
	inst.Progress.TotalChunks = len(inst.ChunkMap)
	inst.Progress.mu.Unlock()
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

func (inst *Installer) EnumerateChunksWithFileOrder() []*ChunkMetaData {
	positiveSubstrs := []string{"globalgame", "pkg_version", "data.unity3d", "exe"}
	negativeSubstrs := []string{"/"}

	// Helper function to calculate priority
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

	// Struct to hold file and its priority
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

// ----- functions for locking progress updates -----

func (p *InstallProgress) IncrementDownloadedBytes(n int64) {
	p.mu.Lock()
	p.DownloadedBytes += n
	p.mu.Unlock()
}

func (p *InstallProgress) IncrementDownloadedChunks() {
	p.mu.Lock()
	p.DownloadedChunks++
	p.mu.Unlock()
}

func (p *InstallProgress) IncrementTotalBytes(n int64) {
	p.mu.Lock()
	p.TotalBytes += n
	p.mu.Unlock()
}

func (p *InstallProgress) IncrementDecompressedChunks() {
	p.mu.Lock()
	p.DecompressedChunks++
	p.mu.Unlock()
}

func (p *InstallProgress) IncrementVerifiedChunks() {
	p.mu.Lock()
	p.VerifiedChunks++
	p.mu.Unlock()
}

func (p *InstallProgress) IncrementAssembledChunks() {
	p.mu.Lock()
	p.AssembledChunks++
	p.mu.Unlock()
}

func (p *InstallProgress) IncrementVerifiedFiles() {
	p.mu.Lock()
	p.VerifiedFiles++
	p.mu.Unlock()
}
