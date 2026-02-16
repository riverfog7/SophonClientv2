package decompressor

import (
	"SophonClientv2/internal/config"
	"SophonClientv2/internal/logging"
	"SophonClientv2/pkg/utils"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
)

func (z *zstdReadCloser) Close() error {
	z.Decoder.Close()
	return z.source.Close()
}

func NewWorker(id int, inputQueue chan DecompressorInput, outputQueue chan DecompressorOutput, wg *sync.WaitGroup) *DecompressorWorker {
	return &DecompressorWorker{
		Id:          id,
		InputQueue:  inputQueue,
		OutputQueue: outputQueue,
		wg:          wg,
	}
}

func (worker *DecompressorWorker) Start() {
	logging.GlobalLogger.Debug("Started decompressor worker " + strconv.Itoa(worker.Id))

	worker.wg.Add(1)
	go func() {
		defer worker.wg.Done()
		for input := range worker.InputQueue {
			if input.Content == nil {
				err := fmt.Errorf("nil compressed content")
				logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": " + err.Error())
				worker.OutputQueue <- DecompressorOutput{Content: nil, Suceeded: false, Err: err, Payload: input.Payload}
				continue
			}

			dec, err := zstd.NewReader(input.Content)
			if err != nil {
				utils.CloseQuietly(input.Content)
				logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Failed to create zstd reader: " + err.Error())
				worker.OutputQueue <- DecompressorOutput{Content: nil, Suceeded: false, Err: err, Payload: input.Payload}
				continue
			}

			logging.GlobalLogger.Debug("Worker " + strconv.Itoa(worker.Id) + ": Successfully decompressed content")
			worker.OutputQueue <- DecompressorOutput{Content: &zstdReadCloser{Decoder: dec, source: input.Content}, Suceeded: true, Payload: input.Payload}
		}
	}()
}

func NewDecompressor(buffSize int) *Decompressor {
	logging.GlobalLogger.Info("Initializing Decompressor with " + strconv.Itoa(config.Config.CocurrentDecompressions) + " workers")

	threadCount := config.Config.CocurrentDecompressions
	inputQueue := make(chan DecompressorInput, buffSize)
	outputQueue := make(chan DecompressorOutput, buffSize)
	workers := make([]*DecompressorWorker, threadCount)
	wg := &sync.WaitGroup{}

	for i := 0; i < threadCount; i++ {
		workers[i] = NewWorker(i, inputQueue, outputQueue, wg)
		workers[i].Start()
	}

	decompressor := &Decompressor{
		ThreadCount:  threadCount,
		InputQueue:   inputQueue,
		OutputQueue:  outputQueue,
		Workers:      workers,
		wg:           wg,
		statusStopCh: make(chan struct{}),
	}
	decompressor.StartPrintChannelStatus(config.Config.QueueLengthPrintInterval)
	return decompressor
}

func (d *Decompressor) StartPrintChannelStatus(intervalSeconds int) {
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

func (d *Decompressor) PrintChannelStatus() {
	logging.GlobalLogger.Debug("Decompressor Input Queue Length: " + strconv.Itoa(len(d.InputQueue)) + "/" + strconv.Itoa(cap(d.InputQueue)))
	logging.GlobalLogger.Debug("Decompressor Output Queue Length: " + strconv.Itoa(len(d.OutputQueue)) + "/" + strconv.Itoa(cap(d.OutputQueue)))
}

func (d *Decompressor) Stop() {
	d.stopOnce.Do(func() {
		close(d.statusStopCh)
		close(d.InputQueue)
		d.wg.Wait()
		close(d.OutputQueue)
		logging.GlobalLogger.Info("Decompressor stopped")
	})
}

func (d *Decompressor) EnqueueDecompression(content io.ReadCloser, payload any) {
	utils.NonBlockingEnqueue(d.InputQueue, DecompressorInput{Content: content, Payload: payload})
}

func (d *Decompressor) GetOutputChannel() chan DecompressorOutput {
	return d.OutputQueue
}
