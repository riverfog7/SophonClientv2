package installer

import "SophonClientv2/internal/logging"

// Start starts all pipeline stages and begins processing
func (inst *Installer) Start() {
	logging.GlobalLogger.Info("Starting installation pipeline")

	// Start all pipeline stages (each spawns its own goroutine)
	inst.EnqueueChunks()    // Sends initial chunks to InputQueue
	inst.DownloadChunks()   // InputQueue -> Downloader
	inst.DecompressChunks() // Downloader -> Decompressor
	inst.VerifyChunks()     // Decompressor -> Verifier
	inst.AssembleChunks()   // Verifier -> Assembler
	inst.VerifyFiles()      // Assembler -> Verifier2
	inst.MoveFiles()        // Verifier2 -> Final files (closes InputQueue when all files complete)

	logging.GlobalLogger.Info("All pipeline stages started")
}

// Stop forcefully shuts down the pipeline (use for cancellation/errors)
func (inst *Installer) Stop() {
	logging.GlobalLogger.Info("Stopping installation pipeline")

	// Close InputQueue to signal no more work (may already be closed if installation completed)
	defer func() {
		if r := recover(); r != nil {
			// InputQueue already closed, that's fine
		}
	}()
	close(inst.InputQueue)

	// Stop all components (they will close their output channels)
	inst.Downloader.Stop()
	inst.Decompressor.Stop()
	inst.Verifier.Stop()
	inst.Assembler.Stop()
	inst.Verifier2.Stop()

	// Wait for all pipeline stage goroutines to finish
	inst.wg.Wait()

	logging.GlobalLogger.Info("Installation pipeline stopped")
}

// Wait blocks until the installation completes naturally (all files verified and moved)
// The pipeline will shut down automatically when all files are complete
func (inst *Installer) Wait() {
	logging.GlobalLogger.Info("Waiting for installation to complete")

	// Wait for all pipeline goroutines to finish
	// They will finish after:
	// 1. MoveFiles() closes InputQueue when all files are done
	// 2. Each stage closes its downstream component's output channel
	// 3. All goroutines exit their range loops
	inst.wg.Wait()

	logging.GlobalLogger.Info("Installation completed successfully")
}
