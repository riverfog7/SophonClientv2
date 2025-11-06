package decompressor

import (
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
)

type DecompressorInput struct {
	Content io.ReadCloser
	Payload any
}

type DecompressorOutput struct {
	Content  io.ReadCloser
	Suceeded bool
	Payload  any
}

type DecompressorWorker struct {
	Id          int
	InputQueue  chan DecompressorInput
	OutputQueue chan DecompressorOutput
	wg          *sync.WaitGroup
}

type Decompressor struct {
	ThreadCount int
	InputQueue  chan DecompressorInput
	OutputQueue chan DecompressorOutput
	Workers     []*DecompressorWorker
	wg          *sync.WaitGroup
}

type zstdReadCloser struct {
	*zstd.Decoder
	source io.ReadCloser
}
