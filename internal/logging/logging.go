package logging

import (
	"SophonClientv2/internal/config"
	"fmt"
	"os"
	"path/filepath"
)

type Logger struct {
	LogToFile bool
	LogFile   *os.File // Optional, used if LogToFile is true
}

func NewLogger() *Logger {
	if config.Config.SophonLogToFile {
		filePath := config.Config.SophonLogFile
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			panic(err)
		}

		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		return &Logger{
			LogToFile: true,
			LogFile:   file,
		}
	}

	return &Logger{
		LogToFile: false,
		LogFile:   nil,
	}
}

func (l *Logger) HandleMessage(message string) {
	if l.LogToFile {
		_, err := l.LogFile.WriteString(message + "\n")
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println(message)
	}

	return
}

func (l *Logger) Debug(message string) {
	if config.Config.SophonLogLevel > config.Debug {
		return
	}
	debugPrefix := "[DEBUG] "
	if l.LogToFile {
		l.HandleMessage(debugPrefix + message)
	} else {
		debugColored := "\033[34m" + debugPrefix + message + "\033[0m"
		l.HandleMessage(debugColored)
	}
}

func (l *Logger) Info(message string) {
	if config.Config.SophonLogLevel > config.Info {
		return
	}
	infoPrefix := "[INFO] "
	if l.LogToFile {
		l.HandleMessage(infoPrefix + message)
	} else {
		infoColored := "\033[32m" + infoPrefix + message + "\033[0m"
		l.HandleMessage(infoColored)
	}
}

func (l *Logger) Warn(message string) {
	if config.Config.SophonLogLevel > config.Warn {
		return
	}
	warnPrefix := "[WARN] "
	if l.LogToFile {
		l.HandleMessage(warnPrefix + message)
	} else {
		warnColored := "\033[33m" + warnPrefix + message + "\033[0m"
		l.HandleMessage(warnColored)
	}
}

func (l *Logger) Error(message string) {
	if config.Config.SophonLogLevel > config.Error {
		return
	}
	errorPrefix := "[ERROR] "
	if l.LogToFile {
		l.HandleMessage(errorPrefix + message)
	} else {
		errorColored := "\033[31m" + errorPrefix + message + "\033[0m"
		l.HandleMessage(errorColored)
	}
}

func (l *Logger) Fatal(message string) {
	fatalPrefix := "[FATAL] "
	if l.LogToFile {
		l.HandleMessage(fatalPrefix + message)
	} else {
		fatalColored := "\033[35m" + fatalPrefix + message + "\033[0m"
		l.HandleMessage(fatalColored)
	}

	err := l.LogFile.Sync()
	if err != nil {
		panic(err)
	}
	err = l.LogFile.Close()
	if err != nil {
		panic(err)
	}
	os.Exit(1) // Exit in fatal errors
}

var GlobalLogger = NewLogger()
