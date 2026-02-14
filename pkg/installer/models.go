package installer

import (
	"SophonClientv2/pkg/assembler"
	"SophonClientv2/pkg/decompressor"
	"SophonClientv2/pkg/downloader"
	"SophonClientv2/pkg/verifier"
	"sync"
	"time"
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

type retryDispatchInput struct {
	Metadata *ChunkMetaData
	Stage    string
	Attempt  int
}

type Installer struct {
	GameDir    string
	StagingDir string

	ChunkMap map[string]*ChunkMetaData
	FileMap  map[string]*FileMetaData
	Progress InstallProgress

	InputQueue         chan ChunksInput
	retryDispatchQueue chan retryDispatchInput
	retryOverflowQueue chan retryDispatchInput

	retryOverflowBufferMu sync.Mutex
	retryOverflowBuffer   []retryDispatchInput

	retrySatMu     sync.Mutex
	retrySatSince  time.Time
	retrySatWarned bool

	inputQueueStateMu sync.RWMutex
	inputQueueClosed  bool

	Downloader   *downloader.Downloader
	Decompressor *decompressor.Decompressor
	Verifier     *verifier.Verifier // For chunk verification
	Assembler    *assembler.Assembler
	Verifier2    *verifier.Verifier // For file verification

	wg sync.WaitGroup

	retryMu                 sync.Mutex
	chunkRetryCounts        map[string]int
	fileRetryCounts         map[string]int
	maxChunkPipelineRetries int
	maxFileRebuildRetries   int

	terminalErrMu   sync.RWMutex
	terminalErr     error
	terminalErrOnce sync.Once

	completionMu              sync.Mutex
	completedFiles            map[string]struct{}
	inFlightFileVerifications map[string]struct{}

	inputQueueCloseOnce sync.Once
	stopOnce            sync.Once
}
