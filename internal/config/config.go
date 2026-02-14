package config

import (
	"os"
	"strings"
)

type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Warn
	Error
	Fatal
)

type SophonClientConfig struct {
	MaxManifestDownloadRetries int
	MaxChunkDownloadRetries    int
	MaxChunkPipelineRetries    int
	MaxFileRebuildRetries      int

	DownloaderRetryBaseDelayMs int
	DownloaderRetryMaxDelayMs  int
	DownloaderRetryJitterMs    int

	PipelineRetryBaseDelayMs int
	PipelineRetryMaxDelayMs  int
	PipelineRetryJitterMs    int

	RetryDispatchWorkers     int
	RetryDispatchQueueSize   int
	RetryOverflowQueueSize   int
	RetryOverflowBufferLimit int
	RetrySaturationGraceMs   int

	DownloadChanSize   int
	VerifyChanSize     int
	DecompressChanSize int

	CocurrentDownloads      int
	CocurrentDecompressions int
	CocurrentHashchecks     int

	QueueLengthPrintInterval int

	SophonLogLevel  LogLevel
	SophonLogFile   string
	SophonLogToFile bool
}

func NewSophonClientConfig() SophonClientConfig {
	cfg := SophonClientConfig{
		MaxManifestDownloadRetries: 5,
		MaxChunkDownloadRetries:    5,
		MaxChunkPipelineRetries:    6,
		MaxFileRebuildRetries:      2,

		DownloaderRetryBaseDelayMs: 250,
		DownloaderRetryMaxDelayMs:  5000,
		DownloaderRetryJitterMs:    120,

		PipelineRetryBaseDelayMs: 150,
		PipelineRetryMaxDelayMs:  4000,
		PipelineRetryJitterMs:    100,

		RetryDispatchWorkers:     4,
		RetryDispatchQueueSize:   512,
		RetryOverflowQueueSize:   2048,
		RetryOverflowBufferLimit: 16384,
		RetrySaturationGraceMs:   15000,

		DownloadChanSize:   32,
		VerifyChanSize:     32,
		DecompressChanSize: 32,

		CocurrentDownloads:      16,
		CocurrentDecompressions: 4,
		CocurrentHashchecks:     8,

		QueueLengthPrintInterval: 1,

		SophonLogLevel:  Debug,
		SophonLogFile:   "",
		SophonLogToFile: false,
	}

	if file := os.Getenv("SOPHON_LOG"); file != "" {
		cfg.SophonLogToFile = true
		cfg.SophonLogFile = file
	}
	if lvl := os.Getenv("SOPHON_LOG_LEVEL"); lvl != "" {
		switch strings.ToLower(lvl) {
		case "debug":
			cfg.SophonLogLevel = Debug
		case "info":
			cfg.SophonLogLevel = Info
		case "warn", "warning":
			cfg.SophonLogLevel = Warn
		case "error":
			cfg.SophonLogLevel = Error
		case "fatal":
			cfg.SophonLogLevel = Fatal
		}
	}
	return cfg
}

var Config SophonClientConfig = NewSophonClientConfig()
