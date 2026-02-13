package downloader

import (
	"SophonClientv2/internal/config"
	"SophonClientv2/internal/logging"
	"SophonClientv2/pkg/utils"
	"bytes"
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
			for attempt := 1; attempt <= maxRetries; attempt++ {
				resp, err := worker.HttpClient.Get(input.Url)
				if err != nil {
					if attempt < maxRetries {
						logging.GlobalLogger.Warn("Worker " + strconv.Itoa(worker.Id) + ": Failed to download chunk, retrying... (attempt " + strconv.Itoa(attempt) + ")")
						continue
					}
					logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Failed to download chunk from " + input.Url + ": " + err.Error())
					worker.OutputQueue <- DownloaderOutput{Content: nil, Suceeded: false, Payload: input.Payload}
					break
				}

				if resp.StatusCode != http.StatusOK {
					utils.DrainAndClose(resp.Body)
					if attempt < maxRetries {
						logging.GlobalLogger.Warn("Worker " + strconv.Itoa(worker.Id) + ": Failed to download chunk with status " + resp.Status + ", retrying... (attempt " + strconv.Itoa(attempt) + ")")
						continue
					}
					logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Failed to download chunk from " + input.Url + " with status " + resp.Status)
					worker.OutputQueue <- DownloaderOutput{Content: nil, Suceeded: false, Payload: input.Payload}
					break
				}

				contentBytes, readErr := io.ReadAll(resp.Body)
				utils.CloseQuietly(resp.Body)
				if readErr != nil {
					if attempt < maxRetries {
						logging.GlobalLogger.Warn("Worker " + strconv.Itoa(worker.Id) + ": Failed to read chunk body, retrying... (attempt " + strconv.Itoa(attempt) + ")")
						continue
					}
					logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Error reading response body from " + input.Url + ": " + readErr.Error())
					worker.OutputQueue <- DownloaderOutput{Content: nil, Suceeded: false, Payload: input.Payload}
					break
				}

				logging.GlobalLogger.Debug("Worker " + strconv.Itoa(worker.Id) + ": Successfully downloaded chunk from " + input.Url)
				worker.OutputQueue <- DownloaderOutput{Content: io.NopCloser(bytes.NewReader(contentBytes)), Suceeded: true, Payload: input.Payload}
				break
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

	downloader := &Downloader{
		ThreadCount:  threadCount,
		HttpClient:   httpClient,
		InputQueue:   inputQueue,
		OutputQueue:  outputQueue,
		Workers:      workers,
		wg:           wg,
		statusStopCh: make(chan struct{}),
	}
	downloader.StartPrintChannelStatus(config.Config.QueueLengthPrintInterval)
	return downloader
}

func (d *Downloader) StartPrintChannelStatus(intervalSeconds int) {
	go func() {
		ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-d.statusStopCh:
				return
			case <-ticker.C:
				d.PrintChannelStatus()
			}
		}
	}()
}

func (d *Downloader) PrintChannelStatus() {
	logging.GlobalLogger.Debug("Downloader InputQueue: " + strconv.Itoa(len(d.InputQueue)) + "/" + strconv.Itoa(cap(d.InputQueue)))
	logging.GlobalLogger.Debug("Downloader OutputQueue: " + strconv.Itoa(len(d.OutputQueue)) + "/" + strconv.Itoa(cap(d.OutputQueue)))
}

func (d *Downloader) Stop() {
	d.stopOnce.Do(func() {
		close(d.statusStopCh)
		close(d.InputQueue)
		d.wg.Wait()
		close(d.OutputQueue)
		logging.GlobalLogger.Info("Downloader stopped")
	})
}

func (d *Downloader) EnqueueDownload(url string, payload any) {
	utils.NonBlockingEnqueue(d.InputQueue, DownloaderInput{Url: url, Payload: payload})
}

func (d *Downloader) GetOutputChannel() chan DownloaderOutput {
	return d.OutputQueue
}
