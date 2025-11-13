package assembler

import (
	"SophonClientv2/internal/config"
	"SophonClientv2/internal/logging"
	"SophonClientv2/pkg/utils"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

func NewAssembler(stagingDir string, buffSize int) *Assembler {
	inputQueue := make(chan AssemblerInput, buffSize)
	outputQueue := make(chan AssemblerOutput, buffSize)
	wg := &sync.WaitGroup{}

	asm := &Assembler{
		StagingDir:  stagingDir,
		InputQueue:  inputQueue,
		OutputQueue: outputQueue,
		wg:          wg,
	}

	asm.Start() // Not multi threaded
	asm.StartPrintChannelStatus(config.Config.QueueLengthPrintInterval)

	return asm
}

func (a *Assembler) StartPrintChannelStatus(intervalSeconds int) {
	go func() {
		for {
			if a.wg == nil {
				return
			}
			a.PrintChannelStatus()
			<-time.After(time.Duration(intervalSeconds) * time.Second)
		}
	}()
}

func (a *Assembler) PrintChannelStatus() {
	logging.GlobalLogger.Debug("Assembler Input Queue Length: " + strconv.Itoa(len(a.InputQueue)) + "/" + strconv.Itoa(cap(a.InputQueue)))
	logging.GlobalLogger.Debug("Assembler Output Queue Length: " + strconv.Itoa(len(a.OutputQueue)) + "/" + strconv.Itoa(cap(a.OutputQueue)))
}

func (a *Assembler) Start() {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		for input := range a.InputQueue {
			fullPath := filepath.Join(a.StagingDir, input.FilePath)

			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				logging.GlobalLogger.Error(fmt.Sprintf("Failed to create directory %s: %v", dir, err))
				utils.CloseStreamSafe(input.Content)
				a.OutputQueue <- AssemblerOutput{FilePath: input.FilePath, ChunkID: input.ChunkID, Succeeded: false, Payload: input.Payload}
				continue
			}

			file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				logging.GlobalLogger.Error(fmt.Sprintf("Failed to open file %s: %v", fullPath, err))
				utils.CloseStreamSafe(input.Content)
				a.OutputQueue <- AssemblerOutput{FilePath: input.FilePath, ChunkID: input.ChunkID, Succeeded: false, Payload: input.Payload}
				continue
			}

			if _, err := file.Seek(int64(input.Offset), io.SeekStart); err != nil {
				logging.GlobalLogger.Error(fmt.Sprintf("Failed to seek to offset %d: %v", input.Offset, err))
				utils.CloseStreamSafe(file)
				utils.CloseStreamSafe(input.Content)
				a.OutputQueue <- AssemblerOutput{FilePath: input.FilePath, ChunkID: input.ChunkID, Succeeded: false, Payload: input.Payload}
				continue
			}

			written, err := io.Copy(file, input.Content)
			utils.CloseStreamSafe(file)
			utils.CloseStreamSafe(input.Content)

			if err != nil {
				logging.GlobalLogger.Error(fmt.Sprintf("Failed to write chunk %s: %v", input.ChunkID, err))
				a.OutputQueue <- AssemblerOutput{FilePath: input.FilePath, ChunkID: input.ChunkID, Succeeded: false, Payload: input.Payload}
				continue
			}

			logging.GlobalLogger.Debug(fmt.Sprintf("Wrote chunk %s to %s at offset %d (%d bytes)", input.ChunkID, input.FilePath, input.Offset, written))
			a.OutputQueue <- AssemblerOutput{FilePath: input.FilePath, ChunkID: input.ChunkID, Succeeded: true, Payload: input.Payload}
		}
	}()
}

func (a *Assembler) Stop() {
	close(a.InputQueue)
	a.wg.Wait()
	a.wg = nil
	close(a.OutputQueue)
	logging.GlobalLogger.Info("Assembler stopped")
}

func (a *Assembler) EnqueueWrite(filePath string, offset uint64, chunkID string, content io.ReadCloser, payload any) {
	input := AssemblerInput{
		FilePath: filePath,
		Offset:   offset,
		ChunkID:  chunkID,
		Content:  content,
		Payload:  payload,
	}

	utils.NonBlockingEnqueue(a.InputQueue, input)
}

func (a *Assembler) GetOutputChannel() chan AssemblerOutput {
	return a.OutputQueue
}
