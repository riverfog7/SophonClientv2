package verifier

import (
	"SophonClientv2/internal/config"
	"SophonClientv2/internal/logging"
	"SophonClientv2/pkg/utils"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"strconv"
	"sync"
)

func NewWorker(id int, inputQueue chan VerifierInput, outputQueue chan VerifierOutput, wg *sync.WaitGroup, returnContent bool) *VerifierWorker {
	return &VerifierWorker{
		Id:            id,
		ReturnContent: returnContent,
		InputQueue:    inputQueue,
		OutputQueue:   outputQueue,
		wg:            wg,
	}
}

func (worker *VerifierWorker) Start() {
	logging.GlobalLogger.Debug("Started verifier worker " + strconv.Itoa(worker.Id))

	worker.wg.Add(1)
	go func() {
		defer worker.wg.Done()
		for input := range worker.InputQueue {
			// Streaming MD5 computation
			hash := md5.New()
			if worker.ReturnContent {
				var buf bytes.Buffer
				teeReader := io.TeeReader(input.Content, &buf) // for passing content (no consume content)
				if _, err := io.Copy(hash, teeReader); err != nil {
					if cerr := input.Content.Close(); cerr != nil {
						logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Error closing content after read failure: " + cerr.Error())
					}
					logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Failed to read content: " + err.Error() + " for " + input.Name)
					logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Marking verification as failed for " + input.Name)
					worker.OutputQueue <- VerifierOutput{Content: nil, Suceeded: false, Payload: input.Payload}
					continue
				}
				if cerr := input.Content.Close(); cerr != nil {
					logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Error closing content after successful read: " + cerr.Error())
				}

				computedHex := hex.EncodeToString(hash.Sum(nil))

				if computedHex != input.ExpectedMD5 {
					logging.GlobalLogger.Warn("Worker " + strconv.Itoa(worker.Id) + ": MD5 mismatch - expected " + input.ExpectedMD5 + ", got " + computedHex + " for " + input.Name)
					worker.OutputQueue <- VerifierOutput{Content: nil, Suceeded: false, Payload: input.Payload}
					continue
				}

				logging.GlobalLogger.Debug("Worker " + strconv.Itoa(worker.Id) + ": MD5 verified successfully for " + input.Name)
				worker.OutputQueue <- VerifierOutput{Content: io.NopCloser(bytes.NewReader(buf.Bytes())), Suceeded: true, Payload: input.Payload}
			} else {
				// No copying content to memory (stream to hash)
				if _, err := io.Copy(hash, input.Content); err != nil {
					if cerr := input.Content.Close(); cerr != nil {
						logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Error closing content after read failure: " + cerr.Error())
					}
					logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Failed to read content: " + err.Error() + " for " + input.Name)
					logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Marking verification as failed for " + input.Name)
					worker.OutputQueue <- VerifierOutput{Content: nil, Suceeded: false, Payload: input.Payload}
					continue
				}
				if cerr := input.Content.Close(); cerr != nil {
					logging.GlobalLogger.Error("Worker " + strconv.Itoa(worker.Id) + ": Error closing content after successful read: " + cerr.Error())
				}

				computedHex := hex.EncodeToString(hash.Sum(nil))

				if computedHex != input.ExpectedMD5 {
					logging.GlobalLogger.Warn("Worker " + strconv.Itoa(worker.Id) + ": MD5 mismatch - expected " + input.ExpectedMD5 + ", got " + computedHex + " for " + input.Name)
					worker.OutputQueue <- VerifierOutput{Content: nil, Suceeded: false, Payload: input.Payload}
					continue
				}

				logging.GlobalLogger.Debug("Worker " + strconv.Itoa(worker.Id) + ": MD5 verified successfully for " + input.Name)
				worker.OutputQueue <- VerifierOutput{Content: nil, Suceeded: true, Payload: input.Payload}
			}
		}
	}()
}

func NewVerifier(buffSize int, returnContent bool) *Verifier {
	logging.GlobalLogger.Info("Initializing Verifier with " + strconv.Itoa(config.Config.CocurrentDownloads) + " workers")

	threadCount := config.Config.CocurrentDownloads
	inputQueue := make(chan VerifierInput, buffSize)
	outputQueue := make(chan VerifierOutput, buffSize)
	workers := make([]*VerifierWorker, threadCount)
	wg := &sync.WaitGroup{}

	for i := 0; i < threadCount; i++ {
		workers[i] = NewWorker(i, inputQueue, outputQueue, wg, returnContent)
		workers[i].Start()
	}

	return &Verifier{
		ThreadCount:   threadCount,
		ReturnContent: returnContent,
		InputQueue:    inputQueue,
		OutputQueue:   outputQueue,
		Workers:       workers,
		wg:            wg,
	}
}

func (v *Verifier) Stop() {
	close(v.InputQueue)
	v.wg.Wait()
	close(v.OutputQueue)
	logging.GlobalLogger.Info("Verifier stopped")
}

func (v *Verifier) EnqueueVerification(name string, content io.ReadCloser, expectedMD5 string, payload any) {
	utils.NonBlockingEnqueue(v.InputQueue, VerifierInput{Name: name, Content: content, ExpectedMD5: expectedMD5, Payload: payload})
}

func (v *Verifier) GetOutputChannel() chan VerifierOutput {
	return v.OutputQueue
}
