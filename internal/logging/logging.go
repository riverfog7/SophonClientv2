package logging

import (
	"fmt"
	"os"
	"path/filepath"
)

type Logger struct {
	LogToFIle bool
	LogFile   *os.File // Optional, used if LogToFile is true
}

func NewLogger() *Logger {
	LogFile := os.Getenv("SOPHON_LOG")
	if LogFile != "" {
		dir := filepath.Dir(LogFile)
		err := os.MkdirAll(dir, 075)
		if err != nil {
			panic(err)
		}

		file, err := os.OpenFile(LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		return &Logger{
			LogToFIle: true,
			LogFile:   file,
		}
	}

	return &Logger{
		LogToFIle: false,
		LogFile:   nil,
	}
}

func (l *Logger) HandleMessage(message string) {
	if l.LogToFIle {
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
	debugPrefix := "[DEBUG] "
	if l.LogToFIle {
		l.HandleMessage(debugPrefix + message)
	} else {
		debugColored := "\033[34m" + debugPrefix + message + "\033[0m"
		l.HandleMessage(debugColored)
	}
}

func (l *Logger) Info(message string) {
	infoPrefix := "[INFO] "
	if l.LogToFIle {
		l.HandleMessage(infoPrefix + message)
	} else {
		infoColored := "\033[32m" + infoPrefix + message + "\033[0m"
		l.HandleMessage(infoColored)
	}
}

func (l *Logger) Warn(message string) {
	warnPrefix := "[WARN] "
	if l.LogToFIle {
		l.HandleMessage(warnPrefix + message)
	} else {
		warnColored := "\033[33m" + warnPrefix + message + "\033[0m"
		l.HandleMessage(warnColored)
	}
}

func (l *Logger) Error(message string) {
	errorPrefix := "[ERROR] "
	if l.LogToFIle {
		l.HandleMessage(errorPrefix + message)
	} else {
		errorColored := "\033[31m" + errorPrefix + message + "\033[0m"
		l.HandleMessage(errorColored)
	}
}

func (l *Logger) Fatal(message string) {
	fatalPrefix := "[FATAL] "
	if l.LogToFIle {
		l.HandleMessage(fatalPrefix + message)
	} else {
		fatalColored := "\033[35m" + fatalPrefix + message + "\033[0m"
		l.HandleMessage(fatalColored)
	}

	os.Exit(1) // Exit in fatal errors
}

var GlobalLogger = NewLogger()
