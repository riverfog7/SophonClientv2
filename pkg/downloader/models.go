package downloader

import (
	"io"
	"net/http"
	"sync"
)

type DownloaderInput struct {
	Url     string
	Payload any
}

type DownloaderOutput struct {
	Content  io.ReadCloser
	Suceeded bool
	Payload  any
}

type DownloaderWorker struct {
	Id          int
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
}
