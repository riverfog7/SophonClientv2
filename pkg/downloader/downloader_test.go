package downloader

import (
	"SophonClientv2/internal/config"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func waitForWorkerStop(t *testing.T, cancel context.CancelFunc, inputQueue chan DownloaderInput, wg *sync.WaitGroup) {
	t.Helper()
	close(inputQueue)
	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for downloader worker to stop")
	}
}

func TestIsRetryableHTTPStatus(t *testing.T) {
	if !isRetryableHTTPStatus(http.StatusRequestTimeout) {
		t.Fatal("expected 408 to be retryable")
	}
	if !isRetryableHTTPStatus(http.StatusTooManyRequests) {
		t.Fatal("expected 429 to be retryable")
	}
	if isRetryableHTTPStatus(http.StatusNotFound) {
		t.Fatal("expected 404 to be non-retryable")
	}
	if isRetryableHTTPStatus(http.StatusConflict) {
		t.Fatal("expected 409 to be non-retryable")
	}
	if isRetryableHTTPStatus(http.StatusGone) {
		t.Fatal("expected 410 to be non-retryable")
	}
	if isRetryableHTTPStatus(http.StatusLocked) {
		t.Fatal("expected 423 to be non-retryable")
	}
}

func TestWorkerMarksHTTP404AsNonRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	inputQueue := make(chan DownloaderInput, 1)
	outputQueue := make(chan DownloaderOutput, 1)
	wg := &sync.WaitGroup{}

	worker := NewWorker(1, ctx, server.Client(), inputQueue, outputQueue, wg)
	worker.Start()

	inputQueue <- DownloaderInput{Url: server.URL, Payload: "chunk"}

	select {
	case out := <-outputQueue:
		if out.Suceeded {
			t.Fatal("expected download to fail for 404")
		}
		if out.Retryable {
			t.Fatal("expected 404 failure to be non-retryable")
		}
		if out.Err == nil {
			t.Fatal("expected error to be attached to failed output")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for downloader output")
	}

	waitForWorkerStop(t, cancel, inputQueue, wg)
}

func TestWorkerMarksRequestBuildFailureAsNonRetryable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	inputQueue := make(chan DownloaderInput, 1)
	outputQueue := make(chan DownloaderOutput, 1)
	wg := &sync.WaitGroup{}

	worker := NewWorker(1, ctx, http.DefaultClient, inputQueue, outputQueue, wg)
	worker.Start()

	inputQueue <- DownloaderInput{Url: "http://[::1", Payload: "chunk"}

	select {
	case out := <-outputQueue:
		if out.Suceeded {
			t.Fatal("expected download to fail for invalid URL")
		}
		if out.Retryable {
			t.Fatal("expected request build failure to be non-retryable")
		}
		if out.Err == nil {
			t.Fatal("expected error to be attached to failed output")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for downloader output")
	}

	waitForWorkerStop(t, cancel, inputQueue, wg)
}

func TestNewDownloaderClampsInvalidConcurrencyAndInterval(t *testing.T) {
	origConcurrency := config.Config.CocurrentDownloads
	origInterval := config.Config.QueueLengthPrintInterval
	defer func() {
		config.Config.CocurrentDownloads = origConcurrency
		config.Config.QueueLengthPrintInterval = origInterval
	}()

	config.Config.CocurrentDownloads = 0
	config.Config.QueueLengthPrintInterval = 0

	d := NewDownloader(1)
	if d.ThreadCount != 1 {
		t.Fatalf("expected thread count to clamp to 1, got: %d", d.ThreadCount)
	}
	if len(d.Workers) != 1 {
		t.Fatalf("expected exactly one worker after clamp, got: %d", len(d.Workers))
	}

	d.Stop()
}
