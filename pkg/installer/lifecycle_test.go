package installer

import (
	"errors"
	"strings"
	"testing"
)

func newRetryHandlingTestInstaller() *Installer {
	return &Installer{
		InputQueue:         make(chan ChunksInput, 4),
		retryDispatchQueue: make(chan retryDispatchInput, 4),
		retryOverflowQueue: make(chan retryDispatchInput, 4),

		chunkRetryCounts:        make(map[string]int),
		fileRetryCounts:         make(map[string]int),
		maxChunkPipelineRetries: 3,
		maxFileRebuildRetries:   1,

		completedFiles:            make(map[string]struct{}),
		inFlightFileVerifications: make(map[string]struct{}),
	}
}

func TestHandleDownloadFailureRetryableRequeuesChunk(t *testing.T) {
	inst := newRetryHandlingTestInstaller()
	cm := &ChunkMetaData{ChunkID: "chunk-retry"}

	inst.handleDownloadFailure(cm, errors.New("temporary timeout"), true)

	if err := inst.getTerminalError(); err != nil {
		t.Fatalf("expected no terminal error for retryable failure, got: %v", err)
	}

	if got := inst.chunkRetryCounts[cm.ChunkID]; got != 1 {
		t.Fatalf("expected retry count to be 1, got: %d", got)
	}

	select {
	case retry := <-inst.retryDispatchQueue:
		if retry.Metadata != cm {
			t.Fatal("expected original chunk metadata to be dispatched for retry")
		}
		if retry.Stage != "download" {
			t.Fatalf("expected retry stage to be download, got: %s", retry.Stage)
		}
	default:
		t.Fatal("expected retry dispatch input to be queued")
	}
}

func TestHandleDownloadFailureNonRetryableSetsTerminalError(t *testing.T) {
	inst := newRetryHandlingTestInstaller()
	cm := &ChunkMetaData{ChunkID: "chunk-terminal"}

	inst.handleDownloadFailure(cm, errors.New("unexpected http status: 404"), false)

	err := inst.getTerminalError()
	if err == nil {
		t.Fatal("expected terminal error for non-retryable failure")
	}
	if !strings.Contains(err.Error(), cm.ChunkID) {
		t.Fatalf("expected terminal error to include chunk id %s, got: %v", cm.ChunkID, err)
	}

	if !inst.isInputQueueClosed() {
		t.Fatal("expected input queue to be closed after terminal error")
	}

	if got := len(inst.retryDispatchQueue); got != 0 {
		t.Fatalf("expected no retry dispatch for non-retryable failure, got: %d", got)
	}
	if got := len(inst.retryOverflowQueue); got != 0 {
		t.Fatalf("expected no retry overflow dispatch for non-retryable failure, got: %d", got)
	}
}
