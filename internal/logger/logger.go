package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/ryan-gang/kindle-send-daemon/internal/config"
)

type LogLevel int

const (
	INFO LogLevel = iota
	WARN
	ERROR
	DEBUG
)

type Logger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	file        *os.File
}

func (l *Logger) Init(cfg config.ConfigProvider) error {
	if cfg == nil {
		return fmt.Errorf("config not provided")
	}

	logDir := filepath.Dir(cfg.GetLogPath())
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	file, err := os.OpenFile(cfg.GetLogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	multiWriter := io.MultiWriter(file, os.Stdout)

	l.infoLogger = log.New(multiWriter, "INFO:  ", log.Ldate|log.Ltime|log.Lshortfile)
	l.warnLogger = log.New(multiWriter, "WARN:  ", log.Ldate|log.Ltime|log.Lshortfile)
	l.errorLogger = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	l.debugLogger = log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	l.file = file

	return nil
}

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) Info(v ...any) {
	if l.infoLogger != nil {
		l.infoLogger.Println(v...)
	}
}

func (l *Logger) Infof(format string, v ...any) {
	if l.infoLogger != nil {
		l.infoLogger.Printf(format, v...)
	}
}

func (l *Logger) Warn(v ...any) {
	if l.warnLogger != nil {
		l.warnLogger.Println(v...)
	}
}

func (l *Logger) Warnf(format string, v ...any) {
	if l.warnLogger != nil {
		l.warnLogger.Printf(format, v...)
	}
}

func (l *Logger) Error(v ...any) {
	if l.errorLogger != nil {
		l.errorLogger.Println(v...)
	}
}

func (l *Logger) Errorf(format string, v ...any) {
	if l.errorLogger != nil {
		l.errorLogger.Printf(format, v...)
	}
}

func (l *Logger) Debug(v ...any) {
	if l.debugLogger != nil {
		l.debugLogger.Println(v...)
	}
}

func (l *Logger) Debugf(format string, v ...any) {
	if l.debugLogger != nil {
		l.debugLogger.Printf(format, v...)
	}
}