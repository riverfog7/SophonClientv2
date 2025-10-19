package operations

import (
	"SophonClientv2/pkg/assembler"
	"SophonClientv2/pkg/decompressor"
	"SophonClientv2/pkg/downloader"
	"SophonClientv2/pkg/verifier"
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

	ChunkMap map[string]*ChunkMetaData
	FileMap  map[string]*FileMetaData
	Progress InstallProgress

	Downloader   *downloader.Downloader
	Decompressor *decompressor.Decompressor
	Verifier     *verifier.Verifier
	Assembler    *assembler.Assembler

	wg sync.WaitGroup
}

func NewInstaller(gameDir, stagingDir string) *Installer {
	return &Installer{
		GameDir:    gameDir,
		StagingDir: stagingDir,

		ChunkMap: make(map[string]*ChunkMetaData),
		FileMap:  make(map[string]*FileMetaData),
		Progress: InstallProgress{},
	}
}
