package downloader

import (
	"SophonClientv2/internal/config"
	"SophonClientv2/internal/logging"
	"SophonClientv2/pkg/utils"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

func downloaderRetryDelay(attempt int) time.Duration {
	if attempt < 1 || config.Config.DownloaderRetryBaseDelayMs <= 0 {
		return 0
	}

	delayMs := config.Config.DownloaderRetryBaseDelayMs
	maxMs := config.Config.DownloaderRetryMaxDelayMs
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

	if config.Config.DownloaderRetryJitterMs > 0 {
		delayMs += rand.Intn(config.Config.DownloaderRetryJitterMs + 1)
	}

	return time.Duration(delayMs) * time.Millisecond
}

func shouldLogRetry(attempt, maxRetries int) bool {
	if maxRetries <= 2 {
		return true
	}
	return attempt == 1 || attempt == maxRetries-1
}

func sleepWithContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func isRetryableHTTPStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func isRetryableDownloadError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if errors.Is(urlErr.Err, context.Canceled) {
			return false
		}
		if errors.Is(urlErr.Err, context.DeadlineExceeded) {
			return true
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	return true
}

func NewWorker(id int, ctx context.Context, httpClient *http.Client, inputQueue chan DownloaderInput, outputQueue chan DownloaderOutput, wg *sync.WaitGroup) *DownloaderWorker {
	return &DownloaderWorker{
		Id:          id,
		Ctx:         ctx,
		HttpClient:  httpClient,
		InputQueue:  inputQueue,
		OutputQueue: outputQueue,
		wg:          wg,
	}
}

func (worker *DownloaderWorker) emitOutput(out DownloaderOutput) bool {
	select {
	case <-worker.Ctx.Done():
		return false
	case worker.OutputQueue <- out:
		return true
	}
}

func (worker *DownloaderWorker) emitFailure(payload any, err error, retryable bool) bool {
	return worker.emitOutput(DownloaderOutput{Content: nil, Suceeded: false, Err: err, Retryable: retryable, Payload: payload})
}

func (worker *DownloaderWorker) Start() {
	logging.GlobalLogger.Debug("Started downloader worker " + strconv.Itoa(worker.Id))

	worker.wg.Add(1)
	go func() {
		defer worker.wg.Done()
		maxRetries := config.Config.MaxChunkDownloadRetries
		if maxRetries < 1 {
			maxRetries = 1
		}

		for {
			select {
			case <-worker.Ctx.Done():
				return
			case input, ok := <-worker.InputQueue:
				if !ok {
					return
				}

				for attempt := 1; attempt <= maxRetries; attempt++ {
					if worker.Ctx.Err() != nil {
						return
					}

					req, reqErr := http.NewRequestWithContext(worker.Ctx, http.MethodGet, input.Url, nil)
					if reqErr != nil {
						logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Failed to build request for " + input.Url + ": " + reqErr.Error())
						if !worker.emitFailure(input.Payload, fmt.Errorf("build request: %w", reqErr), false) {
							return
						}
						break
					}

					resp, err := worker.HttpClient.Do(req)
					if err != nil {
						if worker.Ctx.Err() != nil || errors.Is(err, context.Canceled) {
							return
						}

						retryable := isRetryableDownloadError(err)
						if retryable && attempt < maxRetries {
							if shouldLogRetry(attempt, maxRetries) {
								logging.GlobalLogger.Warn("Worker " + strconv.Itoa(worker.Id) + ": Download transport failure, retrying... (attempt " + strconv.Itoa(attempt) + ")")
							}
							if !sleepWithContext(worker.Ctx, downloaderRetryDelay(attempt)) {
								return
							}
							continue
						}

						logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Failed to download chunk from " + input.Url + " (transport): " + err.Error())
						if !worker.emitFailure(input.Payload, fmt.Errorf("transport: %w", err), retryable) {
							return
						}
						break
					}

					if resp.StatusCode != http.StatusOK {
						statusCode := resp.StatusCode
						utils.DrainAndClose(resp.Body)

						retryable := isRetryableHTTPStatus(statusCode)
						if retryable && attempt < maxRetries {
							if shouldLogRetry(attempt, maxRetries) {
								logging.GlobalLogger.Warn("Worker " + strconv.Itoa(worker.Id) + ": Download HTTP " + strconv.Itoa(statusCode) + ", retrying... (attempt " + strconv.Itoa(attempt) + ")")
							}
							if !sleepWithContext(worker.Ctx, downloaderRetryDelay(attempt)) {
								return
							}
							continue
						}

						logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Failed to download chunk from " + input.Url + " (http_" + strconv.Itoa(statusCode) + ")")
						if !worker.emitFailure(input.Payload, fmt.Errorf("unexpected http status: %d", statusCode), retryable) {
							return
						}
						break
					}

					contentBytes, readErr := io.ReadAll(resp.Body)
					utils.CloseQuietly(resp.Body)
					if readErr != nil {
						if worker.Ctx.Err() != nil || errors.Is(readErr, context.Canceled) {
							return
						}

						retryable := isRetryableDownloadError(readErr)
						if retryable && attempt < maxRetries {
							if shouldLogRetry(attempt, maxRetries) {
								logging.GlobalLogger.Warn("Worker " + strconv.Itoa(worker.Id) + ": Failed to read chunk body, retrying... (attempt " + strconv.Itoa(attempt) + ")")
							}
							if !sleepWithContext(worker.Ctx, downloaderRetryDelay(attempt)) {
								return
							}
							continue
						}

						logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Error reading response body from " + input.Url + " (read_body): " + readErr.Error())
						if !worker.emitFailure(input.Payload, fmt.Errorf("read body: %w", readErr), retryable) {
							return
						}
						break
					}

					logging.GlobalLogger.Debug("Worker " + strconv.Itoa(worker.Id) + ": Successfully downloaded chunk from " + input.Url)
					if !worker.emitOutput(DownloaderOutput{Content: io.NopCloser(bytes.NewReader(contentBytes)), Suceeded: true, Payload: input.Payload}) {
						return
					}
					break
				}
			}
		}
	}()
}

func NewDownloader(buffSize int) *Downloader {
	threadCount := config.Config.CocurrentDownloads
	if threadCount < 1 {
		logging.GlobalLogger.Warn("Configured concurrent downloads is invalid, defaulting to 1")
		threadCount = 1
	}

	logging.GlobalLogger.Info("Initializing Downloader with " + strconv.Itoa(threadCount) + " concurrent downloads")

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

	ctx, cancel := context.WithCancel(context.Background())

	wg := &sync.WaitGroup{}

	for i := 0; i < threadCount; i++ {
		workers[i] = NewWorker(i, ctx, httpClient, inputQueue, outputQueue, wg)
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
		ctx:          ctx,
		cancel:       cancel,
	}
	downloader.StartPrintChannelStatus(config.Config.QueueLengthPrintInterval)
	return downloader
}

func (d *Downloader) StartPrintChannelStatus(intervalSeconds int) {
	if intervalSeconds <= 0 {
		logging.GlobalLogger.Debug("Downloader queue status ticker disabled due to non-positive interval")
		return
	}

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
		if d.cancel != nil {
			d.cancel()
		}
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
