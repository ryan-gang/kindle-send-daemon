package logger

import (
	"github.com/ryan-gang/kindle-send-daemon/internal/config"
)

// LoggerInterface defines the interface for logging
type LoggerInterface interface {
	Info(v ...any)
	Infof(format string, v ...any)
	Warn(v ...any)
	Warnf(format string, v ...any)
	Error(v ...any)
	Errorf(format string, v ...any)
	Debug(v ...any)
	Debugf(format string, v ...any)
	Close() error
}

// NewLogger creates a new logger instance
func NewLogger(cfg config.ConfigProvider) (LoggerInterface, error) {
	logger := &Logger{}
	if err := logger.Init(cfg); err != nil {
		return nil, err
	}
	return logger, nil
}
