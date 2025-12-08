package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Priority levels
type Priority int

const (
	SEVERE Priority = iota // Highest priority - critical errors
	INFO                   // General informational messages
	FINER                  // Fine-grained informational messages
	FINEST                 // Finest-grained informational messages
	DEBUG                  // Debug-level messages
)

var priorityNames = map[Priority]string{
	SEVERE: "SEVERE",
	INFO:   "INFO",
	FINER:  "FINER",
	FINEST: "FINEST",
	DEBUG:  "DEBUG",
}

// Logger represents a logger instance
type Logger struct {
	mu          sync.Mutex
	priority    Priority
	output      io.Writer
	serviceName string
}

// NewLogger creates a new logger instance
func NewLogger(serviceName string, priority Priority) *Logger {
	return &Logger{
		priority:    priority,
		output:      os.Stdout,
		serviceName: serviceName,
	}
}

// SetPriority sets the minimum priority level to log
func (l *Logger) SetPriority(priority Priority) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.priority = priority
}

// SetOutput sets the output writer for logs
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// log writes a log message if the priority is high enough
func (l *Logger) log(priority Priority, format string, args ...interface{}) {
	if priority > l.priority {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	priorityName := priorityNames[priority]
	message := fmt.Sprintf(format, args...)

	logLine := fmt.Sprintf("[%s] [%s] [%s] %s\n", timestamp, priorityName, l.serviceName, message)

	l.output.Write([]byte(logLine))
}

// Severe logs a SEVERE priority message
func (l *Logger) Severe(format string, args ...interface{}) {
	l.log(SEVERE, format, args...)
}

// Info logs an INFO priority message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Finer logs a FINER priority message
func (l *Logger) Finer(format string, args ...interface{}) {
	l.log(FINER, format, args...)
}

// Finest logs a FINEST priority message
func (l *Logger) Finest(format string, args ...interface{}) {
	l.log(FINEST, format, args...)
}

// Debug logs a DEBUG priority message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Severef logs a SEVERE priority message with formatting
func (l *Logger) Severef(format string, args ...interface{}) {
	l.Severe(format, args...)
}

// Infof logs an INFO priority message with formatting
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(format, args...)
}

// Finerf logs a FINER priority message with formatting
func (l *Logger) Finerf(format string, args ...interface{}) {
	l.Finer(format, args...)
}

// Finestf logs a FINEST priority message with formatting
func (l *Logger) Finestf(format string, args ...interface{}) {
	l.Finest(format, args...)
}

// Debugf logs a DEBUG priority message with formatting
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Debug(format, args...)
}

// ParsePriority parses a priority string to Priority type
func ParsePriority(priority string) Priority {
	switch priority {
	case "SEVERE":
		return SEVERE
	case "INFO":
		return INFO
	case "FINER":
		return FINER
	case "FINEST":
		return FINEST
	case "DEBUG":
		return DEBUG
	default:
		return INFO // Default to INFO
	}
}

// Global logger instance
var defaultLogger *Logger
var once sync.Once

// Init initializes the global logger
func Init(serviceName string, priority Priority) {
	once.Do(func() {
		defaultLogger = NewLogger(serviceName, priority)
	})
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if defaultLogger == nil {
		Init("enterprise-api", INFO)
	}
	return defaultLogger
}

// Global convenience functions
func Severe(format string, args ...interface{}) {
	GetLogger().Severe(format, args...)
}

func Info(format string, args ...interface{}) {
	GetLogger().Info(format, args...)
}

func Finer(format string, args ...interface{}) {
	GetLogger().Finer(format, args...)
}

func Finest(format string, args ...interface{}) {
	GetLogger().Finest(format, args...)
}

func Debug(format string, args ...interface{}) {
	GetLogger().Debug(format, args...)
}

func Severef(format string, args ...interface{}) {
	GetLogger().Severef(format, args...)
}

func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

func Finerf(format string, args ...interface{}) {
	GetLogger().Finerf(format, args...)
}

func Finestf(format string, args ...interface{}) {
	GetLogger().Finestf(format, args...)
}

func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}
