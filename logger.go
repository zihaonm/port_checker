package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// Logger handles logging to file and console
type Logger struct {
	file   *os.File
	logger *log.Logger
	mu     sync.Mutex
}

// NewLogger creates a new Logger instance
func NewLogger(filename string) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := log.New(file, "", 0)

	return &Logger{
		file:   file,
		logger: logger,
	}, nil
}

// Log writes a log message with timestamp
func (l *Logger) Log(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMsg := fmt.Sprintf("[%s] %s", timestamp, message)

	// Write to file
	l.logger.Println(logMsg)

	// Also print to console
	fmt.Println(logMsg)
}

// LogStatusChange logs when an endpoint changes status
func (l *Logger) LogStatusChange(endpoint string, isUp bool) {
	status := "DOWN"
	if isUp {
		status = "UP"
	}
	l.Log(fmt.Sprintf("%s is %s", endpoint, status))
}

// LogInfo logs informational messages
func (l *Logger) LogInfo(message string) {
	l.Log(fmt.Sprintf("INFO: %s", message))
}

// LogError logs error messages
func (l *Logger) LogError(message string) {
	l.Log(fmt.Sprintf("ERROR: %s", message))
}

// LogWarning logs warning messages
func (l *Logger) LogWarning(message string) {
	l.Log(fmt.Sprintf("WARNING: %s", message))
}

// Close closes the log file
func (l *Logger) Close() error {
	return l.file.Close()
}
