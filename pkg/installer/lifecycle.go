package installer

import (
	"SophonClientv2/internal/config"
	"SophonClientv2/internal/logging"
	"fmt"
	"math/rand"
	"time"
)

type fileChunkInstance struct {
	Chunk  *ChunkMetaData
	Offset uint64
}

func chunkInstanceKey(chunkID string, offset uint64) string {
	return fmt.Sprintf("%s:%d", chunkID, offset)
}

func (inst *Installer) closeInputQueue() {
	inst.inputQueueCloseOnce.Do(func() {
		inst.inputQueueStateMu.Lock()
		inst.inputQueueClosed = true
		inst.inputQueueStateMu.Unlock()
		close(inst.InputQueue)
	})
}

func (inst *Installer) isInputQueueClosed() bool {
	inst.inputQueueStateMu.RLock()
	defer inst.inputQueueStateMu.RUnlock()
	return inst.inputQueueClosed
}

func retryDelay(attempt, baseMs, maxMs, jitterMs int) time.Duration {
	if attempt < 1 || baseMs <= 0 {
		return 0
	}

	delayMs := baseMs
	if maxMs > 0 {
		for i := 1; i < attempt; i++ {
			if delayMs >= maxMs {
				break
			}
			delayMs *= 2
			if delayMs > maxMs {
				delayMs = maxMs
				break
			}
		}
	}

	if jitterMs > 0 {
		delayMs += rand.Intn(jitterMs + 1)
	}

	return time.Duration(delayMs) * time.Millisecond
}

func (inst *Installer) retryOverflowBufferLen() int {
	inst.retryOverflowBufferMu.Lock()
	defer inst.retryOverflowBufferMu.Unlock()
	return len(inst.retryOverflowBuffer)
}

func (inst *Installer) pushRetryOverflowBuffer(input retryDispatchInput) int {
	inst.retryOverflowBufferMu.Lock()
	inst.retryOverflowBuffer = append(inst.retryOverflowBuffer, input)
	length := len(inst.retryOverflowBuffer)
	inst.retryOverflowBufferMu.Unlock()
	return length
}

func (inst *Installer) popRetryOverflowBuffer() (retryDispatchInput, bool) {
	inst.retryOverflowBufferMu.Lock()
	defer inst.retryOverflowBufferMu.Unlock()

	count := len(inst.retryOverflowBuffer)
	if count == 0 {
		return retryDispatchInput{}, false
	}

	input := inst.retryOverflowBuffer[count-1]
	inst.retryOverflowBuffer = inst.retryOverflowBuffer[:count-1]
	return input, true
}

func (inst *Installer) clearRetrySaturation() {
	inst.retrySatMu.Lock()
	if !inst.retrySatSince.IsZero() {
		elapsed := time.Since(inst.retrySatSince)
		logging.GlobalLogger.Info(fmt.Sprintf("Retry saturation cleared after %s", elapsed.Truncate(time.Millisecond)))
	}
	inst.retrySatSince = time.Time{}
	inst.retrySatWarned = false
	inst.retrySatMu.Unlock()
}

func (inst *Installer) markRetrySaturation(stage string, buffered int) time.Duration {
	inst.retrySatMu.Lock()
	defer inst.retrySatMu.Unlock()

	if inst.retrySatSince.IsZero() {
		inst.retrySatSince = time.Now()
	}

	if !inst.retrySatWarned {
		logging.GlobalLogger.Warn(fmt.Sprintf(
			"Retry saturation started at %s (dispatch=%d/%d overflow=%d/%d buffered=%d)",
			stage,
			len(inst.retryDispatchQueue), cap(inst.retryDispatchQueue),
			len(inst.retryOverflowQueue), cap(inst.retryOverflowQueue),
			buffered,
		))
		inst.retrySatWarned = true
	}

	return time.Since(inst.retrySatSince)
}

func (inst *Installer) enqueueRetryDispatch(cm *ChunkMetaData, stage string, attempt int) (ok bool) {
	if cm == nil {
		inst.setTerminalError(fmt.Errorf("nil chunk metadata for retry dispatch at %s", stage))
		return false
	}
	if inst.hasTerminalError() || inst.isInputQueueClosed() {
		return false
	}

	defer func() {
		if recover() != nil {
			ok = false
		}
	}()

	input := retryDispatchInput{Metadata: cm, Stage: stage, Attempt: attempt}

	select {
	case inst.retryDispatchQueue <- input:
		inst.clearRetrySaturation()
		return true
	default:
	}

	select {
	case inst.retryOverflowQueue <- input:
		inst.clearRetrySaturation()
		return true
	default:
	}

	buffered := inst.pushRetryOverflowBuffer(input)
	limit := config.Config.RetryOverflowBufferLimit
	if limit > 0 && buffered > limit {
		elapsed := inst.markRetrySaturation(stage, buffered)
		grace := time.Duration(config.Config.RetrySaturationGraceMs) * time.Millisecond
		if grace <= 0 {
			grace = 15 * time.Second
		}
		if elapsed >= grace {
			inst.setTerminalError(fmt.Errorf("retry saturation exceeded grace at %s (buffered=%d, limit=%d, elapsed=%s)", stage, buffered, limit, elapsed.Truncate(time.Millisecond)))
			return false
		}
	} else {
		inst.clearRetrySaturation()
	}

	return true
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

func (inst *Installer) isFileCompleted(filePath string) bool {
	inst.completionMu.Lock()
	defer inst.completionMu.Unlock()
	_, ok := inst.completedFiles[filePath]
	return ok
}

func (inst *Installer) tryMarkFileVerifyInFlight(filePath string) bool {
	if filePath == "" {
		return false
	}

	inst.completionMu.Lock()
	defer inst.completionMu.Unlock()

	if _, ok := inst.completedFiles[filePath]; ok {
		return false
	}
	if _, ok := inst.inFlightFileVerifications[filePath]; ok {
		return false
	}

	inst.inFlightFileVerifications[filePath] = struct{}{}
	return true
}

func (inst *Installer) clearFileVerifyInFlight(filePath string) {
	if filePath == "" {
		return
	}

	inst.completionMu.Lock()
	delete(inst.inFlightFileVerifications, filePath)
	inst.completionMu.Unlock()
}

func (inst *Installer) markFileCompleted(filePath string) (bool, int) {
	if filePath == "" {
		return false, 0
	}

	inst.completionMu.Lock()
	defer inst.completionMu.Unlock()

	delete(inst.inFlightFileVerifications, filePath)
	if _, ok := inst.completedFiles[filePath]; ok {
		return false, len(inst.completedFiles)
	}

	inst.completedFiles[filePath] = struct{}{}
	return true, len(inst.completedFiles)
}

func (inst *Installer) fileChunkInstances(fileMeta *FileMetaData) ([]fileChunkInstance, error) {
	if fileMeta == nil {
		return nil, fmt.Errorf("nil file metadata while resolving chunk instances")
	}
	if fileMeta.FilePath == "" {
		return nil, fmt.Errorf("empty file path while resolving chunk instances")
	}

	instances := make([]fileChunkInstance, 0, len(fileMeta.Chunks))
	seen := make(map[string]struct{}, len(fileMeta.Chunks))

	for _, chunkID := range fileMeta.Chunks {
		chunkMeta, ok := inst.ChunkMap[chunkID]
		if !ok {
			return nil, fmt.Errorf("chunk metadata not found for %s (%s)", fileMeta.FilePath, chunkID)
		}

		foundDestination := false
		for _, dest := range chunkMeta.Destinations {
			if dest.File == nil || dest.File.FilePath != fileMeta.FilePath {
				continue
			}

			foundDestination = true
			key := chunkInstanceKey(chunkID, dest.Offset)
			if _, exists := seen[key]; exists {
				continue
			}

			seen[key] = struct{}{}
			instances = append(instances, fileChunkInstance{
				Chunk:  chunkMeta,
				Offset: dest.Offset,
			})
		}

		if !foundDestination {
			return nil, fmt.Errorf("destination not found for file %s in chunk %s", fileMeta.FilePath, chunkID)
		}
	}

	if len(fileMeta.Chunks) > 0 && len(instances) == 0 {
		return nil, fmt.Errorf("no chunk instances resolved for file %s", fileMeta.FilePath)
	}

	return instances, nil
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
	if inst.hasTerminalError() || inst.isInputQueueClosed() {
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
	attempt := count + 1
	inst.chunkRetryCounts[key] = attempt
	inst.retryMu.Unlock()

	if !inst.enqueueRetryDispatch(cm, stage, attempt) {
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
