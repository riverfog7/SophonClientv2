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
	Err      error
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

	statusStopCh chan struct{}
	stopOnce     sync.Once
}

type zstdReadCloser struct {
	*zstd.Decoder
	source io.ReadCloser
}
