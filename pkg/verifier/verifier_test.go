package verifier

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func waitForVerifierWorkerStop(t *testing.T, inputQueue chan VerifierInput, wg *sync.WaitGroup) {
	t.Helper()
	close(inputQueue)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for verifier worker to stop")
	}
}

func TestWorkerPropagatesErrorOnMD5Mismatch(t *testing.T) {
	inputQueue := make(chan VerifierInput, 1)
	outputQueue := make(chan VerifierOutput, 1)
	wg := &sync.WaitGroup{}

	worker := NewWorker(1, inputQueue, outputQueue, wg, false)
	worker.Start()

	inputQueue <- VerifierInput{
		Name:        "chunk-1",
		Content:     io.NopCloser(bytes.NewReader([]byte("test-content"))),
		ExpectedMD5: "00000000000000000000000000000000",
		Payload:     "chunk",
	}

	select {
	case out := <-outputQueue:
		if out.Suceeded {
			t.Fatal("expected verification to fail on md5 mismatch")
		}
		if out.Err == nil {
			t.Fatal("expected verifier error to be propagated")
		}
		if !strings.Contains(out.Err.Error(), "md5 mismatch") {
			t.Fatalf("expected md5 mismatch error, got: %v", out.Err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for verifier output")
	}

	waitForVerifierWorkerStop(t, inputQueue, wg)
}
