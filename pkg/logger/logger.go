package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func ParseLevel(s string) Level {
	switch s {
	case "debug", "DEBUG":
		return DebugLevel
	case "info", "INFO":
		return InfoLevel
	case "warn", "WARN", "warning", "WARNING":
		return WarnLevel
	case "error", "ERROR":
		return ErrorLevel
	case "fatal", "FATAL":
		return FatalLevel
	default:
		return InfoLevel
	}
}

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component,omitempty"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

type Logger struct {
	level      Level
	component  string
	output     io.Writer
	fileOutput *os.File
	format     string
	mu         sync.Mutex
	fields     map[string]interface{}
}

type Config struct {
	Level      string
	Format     string
	OutputFile string
	Component  string
}

var (
	defaultLogger *Logger
	once          sync.Once
)

func Init(cfg Config) error {
	var err error
	once.Do(func() {
		defaultLogger, err = New(cfg)
	})
	return err
}

func New(cfg Config) (*Logger, error) {
	l := &Logger{
		level:     ParseLevel(cfg.Level),
		component: cfg.Component,
		format:    cfg.Format,
		output:    os.Stdout,
		fields:    make(map[string]interface{}),
	}

	if cfg.OutputFile != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.OutputFile), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		f, err := os.OpenFile(cfg.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		l.fileOutput = f
		l.output = io.MultiWriter(os.Stdout, f)
	}

	return l, nil
}

func Default() *Logger {
	if defaultLogger == nil {
		defaultLogger, _ = New(Config{
			Level:  "info",
			Format: "text",
		})
	}
	return defaultLogger
}

func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		level:      l.level,
		component:  component,
		output:     l.output,
		fileOutput: l.fileOutput,
		format:     l.format,
		fields:     copyFields(l.fields),
	}
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newFields := copyFields(l.fields)
	for k, v := range fields {
		newFields[k] = v
	}
	return &Logger{
		level:      l.level,
		component:  l.component,
		output:     l.output,
		fileOutput: l.fileOutput,
		format:     l.format,
		fields:     newFields,
	}
}

func (l *Logger) WithField(key string, value interface{}) *Logger {
	return l.WithFields(map[string]interface{}{key: value})
}

func copyFields(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (l *Logger) log(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	formattedMsg := msg
	if len(args) > 0 {
		formattedMsg = fmt.Sprintf(msg, args...)
	}

	_, file, line, _ := runtime.Caller(2)
	file = filepath.Base(file)

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level.String(),
		Message:   formattedMsg,
		Component: l.component,
		File:      file,
		Line:      line,
		Fields:    l.fields,
	}

	var output string
	if l.format == "json" {
		data, _ := json.Marshal(entry)
		output = string(data)
	} else {
		output = l.formatText(entry)
	}

	fmt.Fprintln(l.output, output)

	if level == FatalLevel {
		os.Exit(1)
	}
}

func (l *Logger) formatText(entry LogEntry) string {
	levelColors := map[string]string{
		"DEBUG": "\033[36m",
		"INFO":  "\033[32m",
		"WARN":  "\033[33m",
		"ERROR": "\033[31m",
		"FATAL": "\033[35m",
	}
	reset := "\033[0m"

	color := levelColors[entry.Level]
	ts := entry.Timestamp[:19]

	base := fmt.Sprintf("%s %s%-5s%s", ts, color, entry.Level, reset)

	if entry.Component != "" {
		base += fmt.Sprintf(" [%s]", entry.Component)
	}

	base += " " + entry.Message

	if len(entry.Fields) > 0 {
		for k, v := range entry.Fields {
			base += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	return base
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(DebugLevel, msg, args...)
}

func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(InfoLevel, msg, args...)
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(WarnLevel, msg, args...)
}

func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(ErrorLevel, msg, args...)
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(FatalLevel, msg, args...)
}

func (l *Logger) Close() error {
	if l.fileOutput != nil {
		return l.fileOutput.Close()
	}
	return nil
}

func Debug(msg string, args ...interface{}) { Default().Debug(msg, args...) }
func Info(msg string, args ...interface{})  { Default().Info(msg, args...) }
func Warn(msg string, args ...interface{})  { Default().Warn(msg, args...) }
func Error(msg string, args ...interface{}) { Default().Error(msg, args...) }
func Fatal(msg string, args ...interface{}) { Default().Fatal(msg, args...) }

func WithComponent(component string) *Logger { return Default().WithComponent(component) }
func WithFields(fields map[string]interface{}) *Logger { return Default().WithFields(fields) }
func WithField(key string, value interface{}) *Logger { return Default().WithField(key, value) }
