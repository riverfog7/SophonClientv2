package downloader

import (
	"context"
	"io"
	"net/http"
	"sync"
)

type DownloaderInput struct {
	Url     string
	Payload any
}

type DownloaderOutput struct {
	Content   io.ReadCloser
	Suceeded  bool
	Err       error
	Retryable bool
	Payload   any
}

type DownloaderWorker struct {
	Id          int
	Ctx         context.Context
	HttpClient  *http.Client
	InputQueue  chan DownloaderInput
	OutputQueue chan DownloaderOutput
	wg          *sync.WaitGroup
}

type Downloader struct {
	ThreadCount int
	HttpClient  *http.Client
	InputQueue  chan DownloaderInput
	OutputQueue chan DownloaderOutput
	Workers     []*DownloaderWorker
	wg          *sync.WaitGroup

	statusStopCh chan struct{}
	stopOnce     sync.Once

	ctx    context.Context
	cancel context.CancelFunc
}
