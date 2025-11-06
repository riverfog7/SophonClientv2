package utils

func NonBlockingEnqueue[T any](ch chan<- T, item T) {
	select {
	case ch <- item:
	default:
		go func() {
			ch <- item
		}()
	}
}
