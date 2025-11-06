package downloader

import (
	"SophonClientv2/internal/config"
	"SophonClientv2/internal/logging"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

func NewWorker(id int, httpClient *http.Client, inputQueue chan DownloaderInput, outputQueue chan DownloaderOutput, wg *sync.WaitGroup) *DownloaderWorker {
	return &DownloaderWorker{
		Id:          id,
		HttpClient:  httpClient,
		InputQueue:  inputQueue,
		OutputQueue: outputQueue,
		wg:          wg,
	}
}

func (worker *DownloaderWorker) Start() {
	logging.GlobalLogger.Debug("Started downloader worker " + strconv.Itoa(worker.Id))

	worker.wg.Add(1)
	go func() {
		defer worker.wg.Done()
		maxRetries := config.Config.MaxChunkDownloadRetries
		for input := range worker.InputQueue {
			var resp *http.Response
			var err error
			for attempt := 1; attempt <= maxRetries; attempt++ {
				resp, err = worker.HttpClient.Get(input.Url)
				if err == nil && resp.StatusCode == http.StatusOK {
					logging.GlobalLogger.Debug("Worker " + strconv.Itoa(worker.Id) + ": Successfully downloaded chunk from " + input.Url)
					worker.OutputQueue <- DownloaderOutput{Content: resp.Body, Suceeded: true, Payload: input.Payload}
					break
				}
				if attempt < maxRetries {
					logging.GlobalLogger.Warn("Worker " + strconv.Itoa(worker.Id) + ": Failed to download chunk, retrying... (attempt " + strconv.Itoa(attempt) + ")")
					continue
				}

				// Cleanup on final failure
				if resp != nil && resp.Body != nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
				if err != nil {
					logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Failed to download chunk from " + input.Url + ": " + err.Error())
				} else {
					logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Failed to download chunk from " + input.Url)
				}
				worker.OutputQueue <- DownloaderOutput{Content: nil, Suceeded: false, Payload: input.Payload}
			}
		}
	}()
}

func NewDownloader(buffSize int) *Downloader {
	logging.GlobalLogger.Info("Initializing Downloader with " + strconv.Itoa(config.Config.CocurrentDownloads) + " concurrent downloads")

	threadCount := config.Config.CocurrentDownloads
	inputQueue := make(chan DownloaderInput, buffSize)
	outputQueue := make(chan DownloaderOutput, buffSize)
	workers := make([]*DownloaderWorker, threadCount)

	transport := &http.Transport{
		MaxIdleConns:        100,              // Maximum idle connections across all hosts
		MaxIdleConnsPerHost: threadCount * 2,  // Maximum idle connections per host
		MaxConnsPerHost:     threadCount * 2,  // Maximum connections per host
		IdleConnTimeout:     90 * time.Second, // How long idle connections stay open
		DisableKeepAlives:   false,            // Enable keep-alive (connection reuse)
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Minute,
	}

	wg := &sync.WaitGroup{}

	for i := 0; i < threadCount; i++ {
		workers[i] = NewWorker(i, httpClient, inputQueue, outputQueue, wg)
		workers[i].Start()
	}

	return &Downloader{
		ThreadCount: threadCount,
		HttpClient:  httpClient,
		InputQueue:  inputQueue,
		OutputQueue: outputQueue,
		Workers:     workers,
		wg:          wg,
	}
}

func (d *Downloader) Stop() {
	close(d.InputQueue)
	d.wg.Wait()
	close(d.OutputQueue)
	logging.GlobalLogger.Info("Downloader stopped")
}

func (d *Downloader) EnqueueDownload(url string, payload any) {
	select {
	case d.InputQueue <- DownloaderInput{Url: url, Payload: payload}:
	default:
		go func() {
			d.InputQueue <- DownloaderInput{Url: url, Payload: payload}
		}()
	}
}

func (d *Downloader) GetOutputChannel() chan DownloaderOutput {
	return d.OutputQueue
}
