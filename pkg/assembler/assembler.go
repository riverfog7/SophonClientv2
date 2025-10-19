package assembler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"SophonClientv2/internal/logging"
)

type AssemblerInput struct {
	FilePath string
	Offset   uint64
	ChunkID  string
	Content  io.ReadCloser
	Payload  any
}

type AssemblerOutput struct {
	ChunkID   string
	Succeeded bool
	Payload   any
}

type Assembler struct {
	StagingDir  string
	InputQueue  chan AssemblerInput
	OutputQueue chan AssemblerOutput
	wg          *sync.WaitGroup
}

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

	return asm
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
				if cerr := input.Content.Close(); cerr != nil {
					logging.GlobalLogger.Warn(fmt.Sprintf("Failed to close content: %v", cerr))
				}
				a.OutputQueue <- AssemblerOutput{ChunkID: input.ChunkID, Succeeded: false, Payload: input.Payload}
				continue
			}

			file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				logging.GlobalLogger.Error(fmt.Sprintf("Failed to open file %s: %v", fullPath, err))
				if cerr := input.Content.Close(); cerr != nil {
					logging.GlobalLogger.Warn(fmt.Sprintf("Failed to close content: %v", cerr))
				}
				a.OutputQueue <- AssemblerOutput{ChunkID: input.ChunkID, Succeeded: false, Payload: input.Payload}
				continue
			}

			if _, err := file.Seek(int64(input.Offset), io.SeekStart); err != nil {
				logging.GlobalLogger.Error(fmt.Sprintf("Failed to seek to offset %d: %v", input.Offset, err))
				if cerr := file.Close(); cerr != nil {
					logging.GlobalLogger.Warn(fmt.Sprintf("Failed to close file: %v", cerr))
				}
				if cerr := input.Content.Close(); cerr != nil {
					logging.GlobalLogger.Warn(fmt.Sprintf("Failed to close content: %v", cerr))
				}
				a.OutputQueue <- AssemblerOutput{ChunkID: input.ChunkID, Succeeded: false, Payload: input.Payload}
				continue
			}

			written, err := io.Copy(file, input.Content)
			if cerr := file.Close(); cerr != nil {
				logging.GlobalLogger.Warn(fmt.Sprintf("Failed to close file: %v", cerr))
			}
			if cerr := input.Content.Close(); cerr != nil {
				logging.GlobalLogger.Warn(fmt.Sprintf("Failed to close content: %v", cerr))
			}

			if err != nil {
				logging.GlobalLogger.Error(fmt.Sprintf("Failed to write chunk %s: %v", input.ChunkID, err))
				a.OutputQueue <- AssemblerOutput{ChunkID: input.ChunkID, Succeeded: false, Payload: input.Payload}
				continue
			}

			logging.GlobalLogger.Debug(fmt.Sprintf("Wrote chunk %s to %s at offset %d (%d bytes)", input.ChunkID, input.FilePath, input.Offset, written))
			a.OutputQueue <- AssemblerOutput{ChunkID: input.ChunkID, Succeeded: true, Payload: input.Payload}
		}
	}()
}

func (a *Assembler) Stop() {
	close(a.InputQueue)
	a.wg.Wait()
	close(a.OutputQueue)
}

func (a *Assembler) EnqueueWrite(filePath string, offset uint64, chunkID string, content io.ReadCloser, payload any) {
	a.InputQueue <- AssemblerInput{
		FilePath: filePath,
		Offset:   offset,
		ChunkID:  chunkID,
		Content:  content,
		Payload:  payload,
	}
}

func (a *Assembler) GetOutputChannel() chan AssemblerOutput {
	return a.OutputQueue
}
