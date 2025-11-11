package verifier

import (
	"io"
	"sync"
)

type VerifierInput struct {
	Name        string
	Content     io.ReadCloser
	ExpectedMD5 string
	Payload     any
}

type VerifierOutput struct {
	Content  io.ReadCloser
	Suceeded bool
	Payload  any
}

type VerifierWorker struct {
	Id            int
	ReturnContent bool
	InputQueue    chan VerifierInput
	OutputQueue   chan VerifierOutput
	wg            *sync.WaitGroup
}

type Verifier struct {
	ReturnContent bool
	ThreadCount   int
	InputQueue    chan VerifierInput
	OutputQueue   chan VerifierOutput
	Workers       []*VerifierWorker
	wg            *sync.WaitGroup
}
