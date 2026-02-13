package installer

import (
	"SophonClientv2/internal/logging"
	"fmt"
)

func (inst *Installer) closeInputQueue() {
	inst.inputQueueCloseOnce.Do(func() {
		close(inst.InputQueue)
	})
}

func (inst *Installer) setTerminalError(err error) {
	if err == nil {
		return
	}

	inst.terminalErrOnce.Do(func() {
		inst.terminalErrMu.Lock()
		inst.terminalErr = err
		inst.terminalErrMu.Unlock()

		logging.GlobalLogger.Error("Installation failed: " + err.Error())
		inst.closeInputQueue()
	})
}

func (inst *Installer) getTerminalError() error {
	inst.terminalErrMu.RLock()
	defer inst.terminalErrMu.RUnlock()
	return inst.terminalErr
}

func (inst *Installer) hasTerminalError() bool {
	return inst.getTerminalError() != nil
}

func (inst *Installer) chunkPayload(payload any, stage string) (*ChunkMetaData, bool) {
	cm, ok := payload.(*ChunkMetaData)
	if !ok || cm == nil {
		inst.setTerminalError(fmt.Errorf("invalid chunk payload at %s: %T", stage, payload))
		return nil, false
	}
	return cm, true
}

func (inst *Installer) filePayload(payload any, stage string) (*FileMetaData, bool) {
	fm, ok := payload.(*FileMetaData)
	if !ok || fm == nil {
		inst.setTerminalError(fmt.Errorf("invalid file payload at %s: %T", stage, payload))
		return nil, false
	}
	return fm, true
}

func (inst *Installer) enqueueChunkInput(input ChunksInput) (ok bool) {
	if inst.hasTerminalError() {
		return false
	}

	defer func() {
		if recover() != nil {
			ok = false
		}
	}()

	inst.InputQueue <- input
	return true
}

func (inst *Installer) chunkRetryKey(cm *ChunkMetaData) string {
	if cm == nil {
		return "<nil>"
	}

	if len(cm.Destinations) == 1 && cm.Destinations[0].File != nil {
		d := cm.Destinations[0]
		return fmt.Sprintf("%s|%s|%d", cm.ChunkID, d.File.FilePath, d.Offset)
	}

	return cm.ChunkID
}

func (inst *Installer) tryRequeueChunk(cm *ChunkMetaData, stage string) bool {
	if cm == nil {
		inst.setTerminalError(fmt.Errorf("nil chunk metadata at %s", stage))
		return false
	}

	key := inst.chunkRetryKey(cm)

	inst.retryMu.Lock()
	count := inst.chunkRetryCounts[key]
	if count >= inst.maxChunkPipelineRetries {
		inst.retryMu.Unlock()
		inst.setTerminalError(fmt.Errorf("chunk retry limit reached for %s at %s", key, stage))
		return false
	}
	inst.chunkRetryCounts[key] = count + 1
	inst.retryMu.Unlock()

	if !inst.enqueueChunkInput(ChunksInput{Metadata: cm}) {
		inst.setTerminalError(fmt.Errorf("failed to re-enqueue chunk %s at %s", key, stage))
		return false
	}

	return true
}

func (inst *Installer) tryRequeueFile(filePath, stage string) bool {
	if filePath == "" {
		inst.setTerminalError(fmt.Errorf("empty file path for requeue at %s", stage))
		return false
	}

	inst.retryMu.Lock()
	count := inst.fileRetryCounts[filePath]
	if count >= inst.maxFileRebuildRetries {
		inst.retryMu.Unlock()
		inst.setTerminalError(fmt.Errorf("file rebuild retry limit reached for %s at %s", filePath, stage))
		return false
	}
	inst.fileRetryCounts[filePath] = count + 1
	inst.retryMu.Unlock()

	return true
}
