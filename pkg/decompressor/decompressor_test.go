package decompressor

import (
	"sync"
	"testing"
	"time"
)

func waitForDecompressorWorkerStop(t *testing.T, inputQueue chan DecompressorInput, wg *sync.WaitGroup) {
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
		t.Fatal("timed out waiting for decompressor worker to stop")
	}
}

func TestWorkerPropagatesErrorOnInvalidCompressedStream(t *testing.T) {
	inputQueue := make(chan DecompressorInput, 1)
	outputQueue := make(chan DecompressorOutput, 1)
	wg := &sync.WaitGroup{}

	worker := NewWorker(1, inputQueue, outputQueue, wg)
	worker.Start()

	inputQueue <- DecompressorInput{Content: nil, Payload: "chunk"}

	select {
	case out := <-outputQueue:
		if out.Suceeded {
			t.Fatal("expected decompression to fail for invalid stream")
		}
		if out.Err == nil {
			t.Fatal("expected decompression error to be propagated")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for decompressor output")
	}

	waitForDecompressorWorkerStop(t, inputQueue, wg)
}
