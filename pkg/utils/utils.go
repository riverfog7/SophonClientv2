package utils

import (
	"SophonClientv2/internal/logging"
	"fmt"
	"io"
)

func NonBlockingEnqueue[T any](ch chan<- T, item T) {
	select {
	case ch <- item:
	default:
		go func() {
			ch <- item
		}()
	}
}

func CloseStreamSafe(stream interface{ Close() error }) {
	if stream == nil {
		return
	}
	_, err := io.Copy(io.Discard, stream.(io.Reader))
	if err != nil {
		logging.GlobalLogger.Warn(fmt.Sprintf("Failed to drain stream before closing: %v", err))
	}

	if err := stream.Close(); err != nil {
		logging.GlobalLogger.Warn(fmt.Sprintf("Failed to close stream: %v", err))
	}
}
