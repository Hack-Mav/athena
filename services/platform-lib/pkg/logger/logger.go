package logger

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus.Logger with additional functionality
type Logger struct {
	*logrus.Logger
	serviceName string
}

// New creates a new logger instance
func New(level, serviceName string) *Logger {
	logger := logrus.New()
	
	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Set formatter
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	// Set output
	logger.SetOutput(os.Stdout)

	return &Logger{
		Logger:      logger,
		serviceName: serviceName,
	}
}

// WithContext adds context information to log entries
func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	entry := l.Logger.WithField("service", l.serviceName)
	
	// Add request ID if available
	if requestID := ctx.Value("request_id"); requestID != nil {
		entry = entry.WithField("request_id", requestID)
	}
	
	// Add user ID if available
	if userID := ctx.Value("user_id"); userID != nil {
		entry = entry.WithField("user_id", userID)
	}
	
	// Add trace ID if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		entry = entry.WithField("trace_id", traceID)
	}
	
	return entry
}

// WithField adds a single field to the log entry
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField("service", l.serviceName).WithField(key, value)
}

// WithFields adds multiple fields to the log entry
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	fields["service"] = l.serviceName
	return l.Logger.WithFields(fields)
}

// WithError adds an error field to the log entry
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithField("service", l.serviceName).WithError(err)
}

// Info logs an info message with service context
func (l *Logger) Info(args ...interface{}) {
	l.Logger.WithField("service", l.serviceName).Info(args...)
}

// Infof logs a formatted info message with service context
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.WithField("service", l.serviceName).Infof(format, args...)
}

// Debug logs a debug message with service context
func (l *Logger) Debug(args ...interface{}) {
	l.Logger.WithField("service", l.serviceName).Debug(args...)
}

// Debugf logs a formatted debug message with service context
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.WithField("service", l.serviceName).Debugf(format, args...)
}

// Warn logs a warning message with service context
func (l *Logger) Warn(args ...interface{}) {
	l.Logger.WithField("service", l.serviceName).Warn(args...)
}

// Warnf logs a formatted warning message with service context
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.WithField("service", l.serviceName).Warnf(format, args...)
}

// Error logs an error message with service context
func (l *Logger) Error(args ...interface{}) {
	l.Logger.WithField("service", l.serviceName).Error(args...)
}

// Errorf logs a formatted error message with service context
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.WithField("service", l.serviceName).Errorf(format, args...)
}

// Fatal logs a fatal message with service context and exits
func (l *Logger) Fatal(args ...interface{}) {
	l.Logger.WithField("service", l.serviceName).Fatal(args...)
}

// Fatalf logs a formatted fatal message with service context and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.WithField("service", l.serviceName).Fatalf(format, args...)
}