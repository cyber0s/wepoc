package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents different log levels
type LogLevel int

const (
	LogLevelTrace LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelTrace:
		return "TRACE"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// EnhancedLogEntry represents a structured log entry
type EnhancedLogEntry struct {
	Timestamp    time.Time              `json:"timestamp"`
	Level        string                 `json:"level"`
	TaskID       int64                  `json:"task_id,omitempty"`
	Component    string                 `json:"component"`
	Message      string                 `json:"message"`
	Error        string                 `json:"error,omitempty"`
	StackTrace   []string               `json:"stack_trace,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
	Command      *CommandInfo           `json:"command,omitempty"`
	Performance  *PerformanceInfo       `json:"performance,omitempty"`
	Environment  *EnvironmentInfo       `json:"environment,omitempty"`
}

// CommandInfo represents command execution information
type CommandInfo struct {
	Executable   string            `json:"executable"`
	Arguments    []string          `json:"arguments"`
	WorkingDir   string            `json:"working_dir"`
	Environment  map[string]string `json:"environment,omitempty"`
	ExitCode     int               `json:"exit_code,omitempty"`
	Duration     time.Duration     `json:"duration,omitempty"`
	OutputSize   int64             `json:"output_size,omitempty"`
}

// PerformanceInfo represents performance metrics
type PerformanceInfo struct {
	MemoryUsage    int64         `json:"memory_usage_bytes"`
	CPUUsage       float64       `json:"cpu_usage_percent"`
	Duration       time.Duration `json:"duration"`
	RequestsPerSec float64       `json:"requests_per_second,omitempty"`
}

// EnvironmentInfo represents system environment information
type EnvironmentInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	GoVersion    string `json:"go_version"`
	WorkingDir   string `json:"working_dir"`
	UserHome     string `json:"user_home"`
	ProcessID    int    `json:"process_id"`
}

// EnhancedLogger provides structured logging with enhanced error tracking
type EnhancedLogger struct {
	taskID      int64
	component   string
	logFile     *os.File
	logDir      string
	minLevel    LogLevel
	context     map[string]interface{}
}

// NewEnhancedLogger creates a new enhanced logger
func NewEnhancedLogger(taskID int64, component string) (*EnhancedLogger, error) {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Create logs directory
	logDir := filepath.Join(homeDir, ".wepoc", "logs", "enhanced")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFileName := fmt.Sprintf("task_%d_%s_%s.log", taskID, component, timestamp)
	logFilePath := filepath.Join(logDir, logFileName)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	logger := &EnhancedLogger{
		taskID:    taskID,
		component: component,
		logFile:   logFile,
		logDir:    logDir,
		minLevel:  LogLevelInfo,
		context:   make(map[string]interface{}),
	}

	// Log initialization
	logger.Info("Enhanced logger initialized", map[string]interface{}{
		"log_file": logFilePath,
		"task_id":  taskID,
		"component": component,
	})

	return logger, nil
}

// SetMinLevel sets the minimum log level
func (el *EnhancedLogger) SetMinLevel(level LogLevel) {
	el.minLevel = level
}

// AddContext adds context information that will be included in all log entries
func (el *EnhancedLogger) AddContext(key string, value interface{}) {
	el.context[key] = value
}

// RemoveContext removes context information
func (el *EnhancedLogger) RemoveContext(key string) {
	delete(el.context, key)
}

// Trace logs a trace message
func (el *EnhancedLogger) Trace(message string, context ...map[string]interface{}) {
	el.log(LogLevelTrace, message, nil, context...)
}

// Debug logs a debug message
func (el *EnhancedLogger) Debug(message string, context ...map[string]interface{}) {
	el.log(LogLevelDebug, message, nil, context...)
}

// Info logs an info message
func (el *EnhancedLogger) Info(message string, context ...map[string]interface{}) {
	el.log(LogLevelInfo, message, nil, context...)
}

// Warn logs a warning message
func (el *EnhancedLogger) Warn(message string, context ...map[string]interface{}) {
	el.log(LogLevelWarn, message, nil, context...)
}

// Error logs an error message with optional error details
func (el *EnhancedLogger) Error(message string, err error, context ...map[string]interface{}) {
	el.log(LogLevelError, message, err, context...)
}

// Fatal logs a fatal error message
func (el *EnhancedLogger) Fatal(message string, err error, context ...map[string]interface{}) {
	el.log(LogLevelFatal, message, err, context...)
}

// LogCommand logs command execution details
func (el *EnhancedLogger) LogCommand(cmd *CommandInfo, message string, context ...map[string]interface{}) {
	entry := el.createLogEntry(LogLevelInfo, message, nil, context...)
	entry.Command = cmd
	el.writeLogEntry(entry)
}

// LogPerformance logs performance metrics
func (el *EnhancedLogger) LogPerformance(perf *PerformanceInfo, message string, context ...map[string]interface{}) {
	entry := el.createLogEntry(LogLevelInfo, message, nil, context...)
	entry.Performance = perf
	el.writeLogEntry(entry)
}

// log is the internal logging method
func (el *EnhancedLogger) log(level LogLevel, message string, err error, context ...map[string]interface{}) {
	if level < el.minLevel {
		return
	}

	entry := el.createLogEntry(level, message, err, context...)
	el.writeLogEntry(entry)

	// Also print to console for important messages
	if level >= LogLevelWarn {
		el.printToConsole(entry)
	}
}

// createLogEntry creates a structured log entry
func (el *EnhancedLogger) createLogEntry(level LogLevel, message string, err error, context ...map[string]interface{}) *EnhancedLogEntry {
	entry := &EnhancedLogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		TaskID:    el.taskID,
		Component: el.component,
		Message:   message,
		Context:   make(map[string]interface{}),
	}

	// Add persistent context
	for k, v := range el.context {
		entry.Context[k] = v
	}

	// Add method-specific context
	for _, ctx := range context {
		for k, v := range ctx {
			entry.Context[k] = v
		}
	}

	// Add error information
	if err != nil {
		entry.Error = err.Error()
		entry.StackTrace = el.getStackTrace()
	}

	// Add environment information for error and fatal logs
	if level >= LogLevelError {
		entry.Environment = el.getEnvironmentInfo()
	}

	return entry
}

// getStackTrace captures the current stack trace
func (el *EnhancedLogger) getStackTrace() []string {
	var stackTrace []string
	
	// Skip the first few frames (this function, log function, etc.)
	for i := 3; i < 15; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		
		// Only include frames from our project
		if strings.Contains(file, "wepoc") {
			stackTrace = append(stackTrace, fmt.Sprintf("%s:%d %s", 
				filepath.Base(file), line, fn.Name()))
		}
	}
	
	return stackTrace
}

// getEnvironmentInfo captures current environment information
func (el *EnhancedLogger) getEnvironmentInfo() *EnvironmentInfo {
	workingDir, _ := os.Getwd()
	userHome, _ := os.UserHomeDir()
	
	return &EnvironmentInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		GoVersion:    runtime.Version(),
		WorkingDir:   workingDir,
		UserHome:     userHome,
		ProcessID:    os.Getpid(),
	}
}

// writeLogEntry writes the log entry to file
func (el *EnhancedLogger) writeLogEntry(entry *EnhancedLogEntry) {
	if el.logFile == nil {
		return
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Printf("❌ Failed to marshal log entry: %v\n", err)
		return
	}

	if _, err := el.logFile.Write(append(data, '\n')); err != nil {
		fmt.Printf("❌ Failed to write log entry: %v\n", err)
	}
}

// printToConsole prints important log entries to console
func (el *EnhancedLogger) printToConsole(entry *EnhancedLogEntry) {
	timestamp := entry.Timestamp.Format("15:04:05")
	
	var icon string
	switch entry.Level {
	case "WARN":
		icon = "⚠️"
	case "ERROR":
		icon = "❌"
	case "FATAL":
		icon = "💀"
	default:
		icon = "ℹ️"
	}
	
	fmt.Printf("%s [%s] %s [%s] %s\n", 
		icon, timestamp, entry.Level, entry.Component, entry.Message)
	
	if entry.Error != "" {
		fmt.Printf("   Error: %s\n", entry.Error)
	}
	
	if len(entry.Context) > 0 {
		fmt.Printf("   Context: %v\n", entry.Context)
	}
}

// Close closes the log file
func (el *EnhancedLogger) Close() error {
	if el.logFile != nil {
		el.Info("Enhanced logger closing")
		return el.logFile.Close()
	}
	return nil
}

// GetLogFilePath returns the path to the current log file
func (el *EnhancedLogger) GetLogFilePath() string {
	if el.logFile != nil {
		return el.logFile.Name()
	}
	return ""
}