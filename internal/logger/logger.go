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

var instance *Logger

func Init() error {
	cfg := config.GetInstance()
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}

	logDir := filepath.Dir(cfg.LogPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	file, err := os.OpenFile(cfg.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	multiWriter := io.MultiWriter(file, os.Stdout)

	instance = &Logger{
		infoLogger:  log.New(multiWriter, "INFO:  ", log.Ldate|log.Ltime|log.Lshortfile),
		warnLogger:  log.New(multiWriter, "WARN:  ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLogger: log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
		debugLogger: log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile),
		file:        file,
	}

	return nil
}

func Close() {
	if instance != nil && instance.file != nil {
		instance.file.Close()
	}
}

func Info(v ...any) {
	if instance != nil {
		instance.infoLogger.Println(v...)
	}
}

func Infof(format string, v ...any) {
	if instance != nil {
		instance.infoLogger.Printf(format, v...)
	}
}

func Warn(v ...any) {
	if instance != nil {
		instance.warnLogger.Println(v...)
	}
}

func Warnf(format string, v ...any) {
	if instance != nil {
		instance.warnLogger.Printf(format, v...)
	}
}

func Error(v ...any) {
	if instance != nil {
		instance.errorLogger.Println(v...)
	}
}

func Errorf(format string, v ...any) {
	if instance != nil {
		instance.errorLogger.Printf(format, v...)
	}
}

func Debug(v ...any) {
	if instance != nil {
		instance.debugLogger.Println(v...)
	}
}

func Debugf(format string, v ...any) {
	if instance != nil {
		instance.debugLogger.Printf(format, v...)
	}
}
