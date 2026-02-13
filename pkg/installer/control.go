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
	inst.stopOnce.Do(func() {
		logging.GlobalLogger.Info("Stopping installation pipeline")

		inst.closeInputQueue()

		inst.Downloader.Stop()
		inst.Decompressor.Stop()
		inst.Verifier.Stop()
		inst.Assembler.Stop()
		inst.Verifier2.Stop()

		inst.wg.Wait()
		logging.GlobalLogger.Info("Installation pipeline stopped")
	})
}

func (inst *Installer) Wait() {
	logging.GlobalLogger.Info("Waiting for installation to complete")
	inst.wg.Wait()
	if err := inst.getTerminalError(); err != nil {
		logging.GlobalLogger.Error("Installation stopped with error: " + err.Error())
		return
	}
	logging.GlobalLogger.Info("Installation completed successfully")
}
