package assembler

import (
	"io"
	"sync"
)

type AssemblerInput struct {
	FilePath string
	Offset   uint64
	ChunkID  string
	Content  io.ReadCloser
	Payload  any
}

type AssemblerOutput struct {
	FilePath  string
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
