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

	CocurrentDownloads      int
	CocurrentDecompressions int
	CocurrentHashchecks     int

	SophonLogLevel  LogLevel
	SophonLogFile   string
	SophonLogToFile bool
}

func NewSophonClientConfig() SophonClientConfig {
	cfg := SophonClientConfig{
		MaxManifestDownloadRetries: 5,
		MaxChunkDownloadRetries:    5,

		CocurrentDownloads:      16,
		CocurrentDecompressions: 4,
		CocurrentHashchecks:     8,

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
