package installer

import "SophonClientv2/internal/logging"

func (inst *Installer) Start() {
	logging.GlobalLogger.Info("Starting installation pipeline")

	inst.EnqueueChunks()
	inst.DownloadChunks()
	inst.DecompressChunks()
	inst.VerifyChunks()
	inst.AssembleChunks()
	inst.VerifyFiles()
	inst.MoveFiles()

	logging.GlobalLogger.Info("All pipeline stages started")
}

func (inst *Installer) Stop() {
	logging.GlobalLogger.Info("Stopping installation pipeline")
	defer func() {
		if r := recover(); r != nil {
		}
	}()
	close(inst.InputQueue)

	inst.Downloader.Stop()
	inst.Decompressor.Stop()
	inst.Verifier.Stop()
	inst.Assembler.Stop()
	inst.Verifier2.Stop()

	inst.wg.Wait()
	logging.GlobalLogger.Info("Installation pipeline stopped")
}

func (inst *Installer) Wait() {
	logging.GlobalLogger.Info("Waiting for installation to complete")
	inst.wg.Wait()
	logging.GlobalLogger.Info("Installation completed successfully")
}
