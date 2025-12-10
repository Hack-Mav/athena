package errors

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// GracefulExit provides a way to exit gracefully with proper error handling
type GracefulExit struct {
	Code    int
	Message string
	Err     error
}

// Error implements the error interface
func (e *GracefulExit) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// NewGracefulExit creates a new graceful exit error
func NewGracefulExit(code int, message string, err error) *GracefulExit {
	return &GracefulExit{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Exit exits the process with the specified code
func (e *GracefulExit) Exit() {
	if e.Err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", e.Error())
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", e.Message)
	}
	os.Exit(e.Code)
}

// Common graceful exit codes
const (
	ExitCodeConfigError  = 1
	ExitCodeDBError      = 2
	ExitCodeServiceError = 3
	ExitCodeNetworkError = 4
)

// NewConfigError creates a configuration error with graceful exit
func NewConfigError(message string, err error) *GracefulExit {
	return NewGracefulExit(ExitCodeConfigError, message, err)
}

// NewDBError creates a database error with graceful exit
func NewDBError(message string, err error) *GracefulExit {
	return NewGracefulExit(ExitCodeDBError, message, err)
}

// NewServiceError creates a service error with graceful exit
func NewServiceError(message string, err error) *GracefulExit {
	return NewGracefulExit(ExitCodeServiceError, message, err)
}

// NewNetworkError creates a network error with graceful exit
func NewNetworkError(message string, err error) *GracefulExit {
	return NewGracefulExit(ExitCodeNetworkError, message, err)
}

// HandleFatalError replaces log.Fatal calls with graceful error handling
func HandleFatalError(message string, err error) {
	gracefulErr := NewGracefulExit(1, message, err)
	gracefulErr.Exit()
}

// HandleConfigError handles configuration errors gracefully
func HandleConfigError(message string, err error) {
	gracefulErr := NewConfigError(message, err)
	gracefulErr.Exit()
}

// HandleDBError handles database errors gracefully
func HandleDBError(message string, err error) {
	gracefulErr := NewDBError(message, err)
	gracefulErr.Exit()
}

// HandleServiceError handles service errors gracefully
func HandleServiceError(message string, err error) {
	gracefulErr := NewServiceError(message, err)
	gracefulErr.Exit()
}

// HandleNetworkError handles network errors gracefully
func HandleNetworkError(message string, err error) {
	gracefulErr := NewNetworkError(message, err)
	gracefulErr.Exit()
}

// SignalHandler handles system signals gracefully
type SignalHandler struct {
	shutdownChan chan os.Signal
	done         chan struct{}
}

// NewSignalHandler creates a new signal handler
func NewSignalHandler() *SignalHandler {
	return &SignalHandler{
		shutdownChan: make(chan os.Signal, 1),
		done:         make(chan struct{}),
	}
}

// Start starts the signal handler
func (sh *SignalHandler) Start() {
	signal.Notify(sh.shutdownChan, syscall.SIGINT, syscall.SIGTERM)
}

// WaitForShutdown waits for shutdown signals
func (sh *SignalHandler) WaitForShutdown() <-chan struct{} {
	go func() {
		sig := <-sh.shutdownChan
		fmt.Printf("Received signal: %v\n", sig)
		close(sh.done)
	}()
	return sh.done
}

// Stop stops the signal handler
func (sh *SignalHandler) Stop() {
	signal.Stop(sh.shutdownChan)
}
