package util

import "fmt"

// ErrorContext provides standardized error formatting for different operations
type ErrorContext string

const (
	ConfigError     ErrorContext = "Config"
	FileError       ErrorContext = "File"
	NetworkError    ErrorContext = "Network"
	ValidationError ErrorContext = "Validation"
	DaemonError     ErrorContext = "Daemon"
	MailError       ErrorContext = "Mail"
	BookmarkError   ErrorContext = "Bookmark"
)

// FormatError creates a standardized error message with context
func FormatError(context ErrorContext, operation string, err error) string {
	return fmt.Sprintf("%s error: %s - %v", context, operation, err)
}

// FormatErrorf creates a standardized error message with context and format
func FormatErrorf(context ErrorContext, operation string, format string, args ...interface{}) string {
	message := fmt.Sprintf(format, args...)
	return fmt.Sprintf("%s error: %s - %s", context, operation, message)
}

// LogError logs an error using the standard format
func LogError(context ErrorContext, operation string, err error) {
	Red.Println(FormatError(context, operation, err))
}

// LogErrorf logs an error using the standard format with formatting
func LogErrorf(context ErrorContext, operation string, format string, args ...interface{}) {
	Red.Println(FormatErrorf(context, operation, format, args...))
}
