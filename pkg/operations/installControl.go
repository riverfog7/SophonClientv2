package operations

import (
	"SophonClientv2/internal/logging"
	"SophonClientv2/pkg/decompressor"
	"SophonClientv2/pkg/downloader"
	"SophonClientv2/pkg/verifier"
	"time"
)

func (inst *Installer) Start() {
	logging.GlobalLogger.Info("Initializing installer components")

	bufferSize := inst.Progress.TotalChunks + 100
	if bufferSize < 100 {
		bufferSize = 100
	}

	inst.Downloader = downloader.NewDownloader(bufferSize)
	inst.Decompressor = decompressor.NewDecompressor(bufferSize)
	inst.Verifier = verifier.NewVerifier(bufferSize)

	logging.GlobalLogger.Info("Starting installation pipeline")

	inst.wg.Add(5)

	go func() {
		defer inst.wg.Done()
		inst.DownloadChunks()
	}()
	time.Sleep(1 * time.Second)
	go func() {
		defer inst.wg.Done()
		inst.DecompressChunks()
	}()
	time.Sleep(1 * time.Second)
	go func() {
		defer inst.wg.Done()
		inst.VerifyChunks()
	}()
	time.Sleep(1 * time.Second)
	go func() {
		defer inst.wg.Done()
		inst.AssembleChunks()
	}()
	time.Sleep(1 * time.Second)
	go func() {
		defer inst.wg.Done()
		inst.VerifyFiles()
	}()

	logging.GlobalLogger.Info("Installation pipeline started")
}

func (inst *Installer) Wait() {
	inst.wg.Wait()
	logging.GlobalLogger.Info("Installation pipeline completed")
}

func (inst *Installer) Stop() {
	logging.GlobalLogger.Info("Stopping installer pipeline")

	if inst.Downloader != nil {
		inst.Downloader.Stop()
	}
	if inst.Decompressor != nil {
		inst.Decompressor.Stop()
	}
	if inst.Verifier != nil {
		inst.Verifier.Stop()
	}
	if inst.Assembler != nil {
		inst.Assembler.Stop()
	}

	logging.GlobalLogger.Info("Installer pipeline stopped")
}
