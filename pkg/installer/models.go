package installer

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
	TotalChunks int
	TotalFiles  int

	DownloadedChunks   int
	DecompressedChunks int
	VerifiedChunks     int
	AssembledChunks    int
	VerifiedFiles      int

	TotalBytes      int64
	DownloadedBytes int64
	mu              sync.RWMutex
}

type ChunksInput struct {
	Metadata *ChunkMetaData
}

type FileOutput struct {
	FilePath string
}

type Installer struct {
	GameDir    string
	StagingDir string

	ChunkMap map[string]*ChunkMetaData
	FileMap  map[string]*FileMetaData
	Progress InstallProgress

	InputQueue chan ChunksInput

	Downloader   *downloader.Downloader
	Decompressor *decompressor.Decompressor
	Verifier     *verifier.Verifier // For chunk verification
	Assembler    *assembler.Assembler
	Verifier2    *verifier.Verifier // For file verification

	wg sync.WaitGroup
}
