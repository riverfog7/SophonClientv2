package installer

import (
	"SophonClientv2/internal/config"
	"SophonClientv2/pkg/assembler"
	"SophonClientv2/pkg/decompressor"
	"SophonClientv2/pkg/downloader"
	"SophonClientv2/pkg/verifier"
)

func NewInstaller(gameDir, stagingDir string, queueSize int) *Installer {
	retryDispatchSize := config.Config.RetryDispatchQueueSize
	if retryDispatchSize <= 0 {
		retryDispatchSize = queueSize * 8
		if retryDispatchSize < 128 {
			retryDispatchSize = 128
		}
	}

	retryOverflowSize := config.Config.RetryOverflowQueueSize
	if retryOverflowSize <= 0 {
		retryOverflowSize = retryDispatchSize * 4
	}

	return &Installer{
		GameDir:    gameDir,
		StagingDir: stagingDir,

		ChunkMap: make(map[string]*ChunkMetaData),
		FileMap:  make(map[string]*FileMetaData),
		Progress: InstallProgress{},

		InputQueue:         make(chan ChunksInput, queueSize),
		retryDispatchQueue: make(chan retryDispatchInput, retryDispatchSize),
		retryOverflowQueue: make(chan retryDispatchInput, retryOverflowSize),

		Downloader:   downloader.NewDownloader(config.Config.DownloadChanSize),
		Decompressor: decompressor.NewDecompressor(config.Config.DecompressChanSize),
		Verifier:     verifier.NewVerifier(config.Config.VerifyChanSize, true),
		Assembler:    assembler.NewAssembler(stagingDir, queueSize),
		Verifier2:    verifier.NewVerifier(config.Config.VerifyChanSize, false),

		chunkRetryCounts:        make(map[string]int),
		chunkRetryLastCause:     make(map[string]retryFailureCause),
		fileRetryCounts:         make(map[string]int),
		fileRetryLastCause:      make(map[string]retryFailureCause),
		maxChunkPipelineRetries: config.Config.MaxChunkPipelineRetries,
		maxFileRebuildRetries:   config.Config.MaxFileRebuildRetries,

		completedFiles:            make(map[string]struct{}),
		inFlightFileVerifications: make(map[string]struct{}),
	}
}
