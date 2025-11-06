package installer

import (
	"SophonClientv2/internal/config"
	"SophonClientv2/pkg/assembler"
	"SophonClientv2/pkg/decompressor"
	"SophonClientv2/pkg/downloader"
	"SophonClientv2/pkg/verifier"
)

func NewInstaller(gameDir, stagingDir string, queueSize int) *Installer {
	return &Installer{
		GameDir:    gameDir,
		StagingDir: stagingDir,

		ChunkMap: make(map[string]*ChunkMetaData),
		FileMap:  make(map[string]*FileMetaData),
		Progress: InstallProgress{},

		InputQueue:  make(chan ChunksInput, queueSize),
		OutputQueue: make(chan FileOutput, queueSize),

		Downloader:   downloader.NewDownloader(config.Config.DownloadChanSize),
		Decompressor: decompressor.NewDecompressor(config.Config.DecompressChanSize),
		Verifier:     verifier.NewVerifier(config.Config.VerifyChanSize),
		Assembler:    assembler.NewAssembler(stagingDir, queueSize),
		Verifier2:    verifier.NewVerifier(config.Config.VerifyChanSize),
	}
}
