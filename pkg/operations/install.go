package operations

// 1. Read Manifest
// 2. Parse manifest (detect chunks that are used multiple times and
//    make a map of chunkID -> (list of where and how to write them))
// 3. For each chunk in the download queue pass it to the downloader
// 4. As chunks are downloaded, pass them to the verifier to check integrity
// 5. If verified, send to assembler to write to disk in all required locations
// 6. If any step fails, retry up to N times before failing the entire install

import (
	"SophonClientv2/internal/logging"
	"SophonClientv2/internal/models"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type ChunkDestination struct {
	File   *FileMetaData
	Offset uint64
}

type ChunkMetaData struct {
	ChunkID          string
	URL              string
	MD5              string
	CompressedSize   uint32
	UncompressedSize uint32
	Destinations     []ChunkDestination
	IsCompressed     bool
}

type FileMetaData struct {
	FilePath string
	Size     int32
	MD5      string
	Chunks   []string
	IsFolder bool
}

type InstallProgress struct {
	TotalChunks      int
	DownloadedChunks int
	VerifiedChunks   int
	AssembledChunks  int
	TotalBytes       int64
	DownloadedBytes  int64
	FailedChunks     []string
	mu               sync.RWMutex
}

type Installer struct {
	GameDir    string
	StagingDir string

	ChunkMap   map[string]*ChunkMetaData
	FileMap    map[string]*FileMetaData
	Progress   InstallProgress
	TaskQueue  chan *ChunkMetaData
	ErrorQueue chan error
	Done       chan struct{}
	wg         sync.WaitGroup
}

func NewInstaller(gameDir, stagingDir string, queueSize int) *Installer {
	return &Installer{
		GameDir:    gameDir,
		StagingDir: stagingDir,

		ChunkMap:   make(map[string]*ChunkMetaData),
		FileMap:    make(map[string]*FileMetaData),
		Progress:   InstallProgress{},
		TaskQueue:  make(chan *ChunkMetaData, queueSize),
		ErrorQueue: make(chan error, queueSize),
		Done:       make(chan struct{}),
	}
}

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
	if err := os.RemoveAll(inst.StagingDir); err != nil {
		// no err on non-existence
		logging.GlobalLogger.Error(fmt.Sprintf("Error clearing staging dir: %v", err))
		return fmt.Errorf("clearing staging dir: %w", err)
	}
	if err := os.MkdirAll(inst.StagingDir, 0o755); err != nil {
		logging.GlobalLogger.Error(fmt.Sprintf("Error creating staging dir: %v", err))
		return fmt.Errorf("creating staging dir: %w", err)
	}

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

		// MD5 hashcheck
		f, err := os.Open(absPath)
		if err != nil {
			logging.GlobalLogger.Fatal(fmt.Sprintf("Error opening existing file %s: %v", absPath, err))
			return fmt.Errorf("opening existing file %s: %w", absPath, err)
		}

		hash := md5.New()
		if _, err := io.Copy(hash, f); err != nil {
			_ = f.Close()
			logging.GlobalLogger.Fatal(fmt.Sprintf("Error hashing file %s: %v", absPath, err))
			return fmt.Errorf("hashing existing file %s: %w", absPath, err)
		}
		_ = f.Close()

		sum := hex.EncodeToString(hash.Sum(nil))
		if sum != fm.MD5 {
			logging.GlobalLogger.Warn(fmt.Sprintf("File failed verification, deleting: %s", absPath))
			if err := os.Remove(absPath); err != nil {
				logging.GlobalLogger.Error(fmt.Sprintf("Error removing invalid file %s: %v", absPath, err))
				return fmt.Errorf("removing invalid file %s: %w", absPath, err)
			}
			continue
		}
		logging.GlobalLogger.Info(fmt.Sprintf("Existing file verified, skipping download: %s", absPath))

		// remove chunks (non duplicates) and file if file is valid
		for _, chunkID := range fm.Chunks {
			cm, ok := inst.ChunkMap[chunkID]
			if !ok {
				continue
			}

			newDests := make([]ChunkDestination, 0, len(cm.Destinations))
			for _, dest := range cm.Destinations {
				if dest.File != fm {
					newDests = append(newDests, dest)
				}
			}

			if len(newDests) == 0 {
				delete(inst.ChunkMap, chunkID)
			} else {
				cm.Destinations = newDests
			}
		}
		delete(inst.FileMap, filePath)
	}

	inst.Progress.TotalChunks = len(inst.ChunkMap)
	inst.ComputeTotalBytes()
	logging.GlobalLogger.Info(fmt.Sprintf("Prepare complete, %d chunks, %d files, total %d bytes remaining", inst.Progress.TotalChunks, len(inst.FileMap), inst.Progress.TotalBytes))
	return nil
}
