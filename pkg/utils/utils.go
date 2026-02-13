package utils

import (
	"SophonClientv2/internal/logging"
	"fmt"
	"io"
)

func NonBlockingEnqueue[T any](ch chan<- T, item T) {
	defer func() {
		if r := recover(); r != nil {
			logging.GlobalLogger.Debug(fmt.Sprintf("Queue send skipped: %v", r))
		}
	}()

	select {
	case ch <- item:
	default:
		ch <- item
	}
}

func CloseQuietly(stream io.Closer) {
	if stream == nil {
		return
	}
	if err := stream.Close(); err != nil {
		logging.GlobalLogger.Warn(fmt.Sprintf("Failed to close stream: %v", err))
	}
}

func DrainAndClose(stream io.ReadCloser) {
	if stream == nil {
		return
	}
	if _, err := io.Copy(io.Discard, stream); err != nil {
		logging.GlobalLogger.Warn(fmt.Sprintf("Failed to drain stream before closing: %v", err))
	}
	if err := stream.Close(); err != nil {
		logging.GlobalLogger.Warn(fmt.Sprintf("Failed to close stream: %v", err))
	}
}
