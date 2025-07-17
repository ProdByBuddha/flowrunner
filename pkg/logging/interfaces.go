// Package logging provides structured logging functionality.
package logging

import (
	"context"
	"time"
)

// Logger provides structured logging
type Logger interface {
	// Debug logs a debug message
	Debug(msg string, fields ...Field)

	// Info logs an info message
	Info(msg string, fields ...Field)

	// Warn logs a warning message
	Warn(msg string, fields ...Field)

	// Error logs an error message
	Error(msg string, fields ...Field)

	// WithFields returns a new logger with the given fields
	WithFields(fields ...Field) Logger

	// WithContext returns a new logger with the given context
	WithContext(ctx context.Context) Logger

	// LogFlowExecution records flow execution events
	LogFlowExecution(flowID string, executionID string, event string, data map[string]interface{})

	// LogNodeExecution records node execution events
	LogNodeExecution(flowID string, executionID string, nodeID string, event string, data map[string]interface{})

	// LogSystemEvent records system-level events
	LogSystemEvent(event string, data map[string]interface{})
}

// Field represents a key-value pair in a log entry
type Field struct {
	// Key is the field name
	Key string

	// Value is the field value
	Value interface{}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	// Timestamp of the log entry
	Timestamp time.Time `json:"timestamp"`

	// Level of the log entry
	Level string `json:"level"`

	// Message is the log message
	Message string `json:"message"`

	// Fields contains additional context
	Fields map[string]interface{} `json:"fields,omitempty"`

	// TraceID for distributed tracing
	TraceID string `json:"trace_id,omitempty"`

	// SpanID for distributed tracing
	SpanID string `json:"span_id,omitempty"`
}

// LogConfig contains configuration for the logger
type LogConfig struct {
	// Level is the minimum log level to output
	Level string `json:"level"`

	// Format is the log format
	Format string `json:"format"`

	// Output is where logs are written
	Output string `json:"output"`

	// FilePath is the path to the log file (if Output is "file")
	FilePath string `json:"file_path,omitempty"`

	// IncludeTimestamp indicates whether to include timestamps
	IncludeTimestamp bool `json:"include_timestamp"`

	// IncludeCaller indicates whether to include caller information
	IncludeCaller bool `json:"include_caller"`
}
