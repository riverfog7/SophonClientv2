package installer

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"testing"
)

func newRetryHandlingTestInstaller() *Installer {
	return &Installer{
		InputQueue:         make(chan ChunksInput, 4),
		retryDispatchQueue: make(chan retryDispatchInput, 4),
		retryOverflowQueue: make(chan retryDispatchInput, 4),

		chunkRetryCounts:        make(map[string]int),
		chunkRetryLastCause:     make(map[string]retryFailureCause),
		fileRetryCounts:         make(map[string]int),
		fileRetryLastCause:      make(map[string]retryFailureCause),
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

func TestTryRequeueChunkRetryLimitIncludesCauseAndHint(t *testing.T) {
	inst := newRetryHandlingTestInstaller()
	inst.maxChunkPipelineRetries = 1

	cm := &ChunkMetaData{
		ChunkID: "chunk-limit",
		Destinations: []ChunkDestination{{
			File: &FileMetaData{FilePath: "data/file.bin"},
		}},
	}
	cause := errors.New("md5 mismatch: expected aa got bb for chunk-limit")

	if ok := inst.tryRequeueChunk(cm, "chunk-verify", cause); !ok {
		t.Fatal("expected first retry enqueue to succeed")
	}

	if ok := inst.tryRequeueChunk(cm, "chunk-verify", cause); ok {
		t.Fatal("expected retry enqueue to fail after reaching retry limit")
	}

	err := inst.TerminalError()
	if err == nil {
		t.Fatal("expected terminal error when chunk retry limit is exceeded")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "chunk retry limit reached") {
		t.Fatalf("expected retry-limit context in terminal error, got: %v", err)
	}
	if !strings.Contains(errMsg, "stage=chunk-verify") {
		t.Fatalf("expected stage context in terminal error, got: %v", err)
	}
	if !strings.Contains(errMsg, "md5 mismatch") {
		t.Fatalf("expected cause context in terminal error, got: %v", err)
	}
	if !strings.Contains(errMsg, "hint:") {
		t.Fatalf("expected actionable hint in terminal error, got: %v", err)
	}
}

func TestTryRequeueFileRetryLimitIncludesCause(t *testing.T) {
	inst := newRetryHandlingTestInstaller()
	inst.maxFileRebuildRetries = 1

	cause := errors.New("md5 mismatch: expected aa got bb for file.bin")

	if ok := inst.tryRequeueFile("data/file.bin", "file-verify", cause); !ok {
		t.Fatal("expected first file requeue to succeed")
	}

	if ok := inst.tryRequeueFile("data/file.bin", "file-verify", cause); ok {
		t.Fatal("expected file requeue to fail after reaching retry limit")
	}

	err := inst.TerminalError()
	if err == nil {
		t.Fatal("expected terminal error when file retry limit is exceeded")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "file rebuild retry limit reached") {
		t.Fatalf("expected file retry-limit context in terminal error, got: %v", err)
	}
	if !strings.Contains(errMsg, "cause=md5 mismatch") {
		t.Fatalf("expected file failure cause in terminal error, got: %v", err)
	}
}

func TestTerminalErrorHintForFilesystemFailure(t *testing.T) {
	inst := newRetryHandlingTestInstaller()
	inst.setTerminalError(fmt.Errorf("write failed: %w", syscall.ENOSPC))

	err := inst.TerminalError()
	if err == nil {
		t.Fatal("expected terminal error to be set")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "hint:") {
		t.Fatalf("expected hint in terminal error, got: %v", err)
	}
	if !strings.Contains(errMsg, "disk space") {
		t.Fatalf("expected disk-space hint in terminal error, got: %v", err)
	}
}

func TestWaitWithErrorReturnsTerminalError(t *testing.T) {
	inst := newRetryHandlingTestInstaller()
	inst.setTerminalError(errors.New("download failed permanently for chunk c1: unexpected http status: 404"))

	err := inst.WaitWithError()
	if err == nil {
		t.Fatal("expected WaitWithError to return terminal error")
	}
	if !strings.Contains(err.Error(), "hint:") {
		t.Fatalf("expected WaitWithError terminal error to include hint, got: %v", err)
	}
}
