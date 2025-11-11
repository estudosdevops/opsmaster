// opsmaster/internal/logger/logger.go
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Configuration holds logger configuration options
type Configuration struct {
	Level  slog.Level
	Format string // "text" or "json"
	Output io.Writer
}

// StructuredContext provides structured context for any application
type StructuredContext struct {
	InstanceID  string
	Operation   string
	Attempt     int
	MaxAttempts int
	Duration    time.Duration
}

// LoggerWithContext extends slog.Logger with structured context methods
type LoggerWithContext struct {
	*slog.Logger
	context StructuredContext
}

var (
	// globalConfig holds the current logger configuration
	globalConfig = &Configuration{
		Level:  slog.LevelInfo, // Safe default for production
		Format: "text",
		Output: os.Stdout,
	}

	// globalLogger holds the singleton logger instance
	globalLogger *slog.Logger

	// mutex protects configuration changes
	mutex sync.RWMutex
)

// parseLogLevel converts string level to slog.Level (DRY principle)
func parseLogLevel(levelStr string) (slog.Level, error) {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN", "WARNING":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid log level: %s (valid: DEBUG, INFO, WARN, ERROR)", levelStr)
	}
}

// parseLogFormat validates and normalizes log format string (DRY principle)
func parseLogFormat(formatStr string) (string, error) {
	switch strings.ToLower(formatStr) {
	case "json":
		return "json", nil
	case "text":
		return "text", nil
	default:
		return "text", fmt.Errorf("invalid log format: %s (valid: json, text)", formatStr)
	}
}

// init initializes logger configuration from environment variables
func init() {
	// Configure log level from environment
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		if level, err := parseLogLevel(levelStr); err != nil {
			// Invalid level, keep default (INFO)
			fmt.Fprintf(os.Stderr, "Invalid LOG_LEVEL '%s', using INFO\n", levelStr)
		} else {
			globalConfig.Level = level
		}
	}

	// Configure log format from environment
	if formatStr := os.Getenv("LOG_FORMAT"); formatStr != "" {
		if format, err := parseLogFormat(formatStr); err != nil {
			// Invalid format, keep default (text)
			fmt.Fprintf(os.Stderr, "Invalid LOG_FORMAT '%s', using text\n", formatStr)
		} else {
			globalConfig.Format = format
		}
	}

	// Initialize logger with configuration
	globalLogger = createLogger(globalConfig)
}

// colorizeLevel applies color formatting to log level string (DRY principle)
func colorizeLevel(level slog.Level, levelStr string) string {
	switch level {
	case slog.LevelDebug:
		return color.MagentaString(levelStr)
	case slog.LevelInfo:
		return color.GreenString(levelStr)
	case slog.LevelWarn:
		return color.YellowString(levelStr)
	case slog.LevelError:
		return color.RedString(levelStr)
	default:
		return levelStr
	}
}

// createLogger creates a new logger instance with the given configuration
func createLogger(config *Configuration) *slog.Logger {
	var handler slog.Handler

	handlerOptions := &slog.HandlerOptions{
		Level:     config.Level,
		AddSource: false,
	}

	if config.Format == "json" {
		// JSON formatter for production
		handler = slog.NewJSONHandler(config.Output, handlerOptions)
	} else {
		// Custom text formatter for development
		handler = &CustomTextHandler{
			level:  config.Level,
			output: config.Output,
		}
	}

	return slog.New(handler)
}

// SetLevel configures the minimum log level
func SetLevel(level string) error {
	mutex.Lock()
	defer mutex.Unlock()

	slogLevel, err := parseLogLevel(level)
	if err != nil {
		return err
	}

	globalConfig.Level = slogLevel
	globalLogger = createLogger(globalConfig)
	return nil
}

// SetFormat configures the log output format
func SetFormat(format string) error {
	mutex.Lock()
	defer mutex.Unlock()

	validFormat, err := parseLogFormat(format)
	if err != nil {
		return err
	}

	globalConfig.Format = validFormat
	globalLogger = createLogger(globalConfig)
	return nil
}

// SetOutput configures the log output destination
func SetOutput(output io.Writer) {
	mutex.Lock()
	defer mutex.Unlock()

	globalConfig.Output = output
	globalLogger = createLogger(globalConfig)
}

// CustomTextHandler é o nosso handler customizado para formatação de texto colorido.
type CustomTextHandler struct {
	level  slog.Level
	output io.Writer
}

// Get retorna uma instância pré-configurada do logger slog com configuração global.
func Get() *slog.Logger {
	mutex.RLock()
	defer mutex.RUnlock()
	return globalLogger
}

// newLoggerWithContext creates a new LoggerWithContext with the given context (DRY principle)
func newLoggerWithContext(ctx StructuredContext) *LoggerWithContext {
	return &LoggerWithContext{
		Logger:  Get(),
		context: ctx,
	}
}

// WithInstance creates a logger with instance context for any application
func WithInstance(instanceID string) *LoggerWithContext {
	return newLoggerWithContext(StructuredContext{
		InstanceID: instanceID,
	})
}

// WithOperation creates a logger with operation context
func WithOperation(operation string) *LoggerWithContext {
	return newLoggerWithContext(StructuredContext{
		Operation: operation,
	})
}

// LoggerWithContext methods for fluent interface
func (l *LoggerWithContext) WithOperation(operation string) *LoggerWithContext {
	l.context.Operation = operation
	return l
}

func (l *LoggerWithContext) WithAttempt(attempt, maxAttempts int) *LoggerWithContext {
	l.context.Attempt = attempt
	l.context.MaxAttempts = maxAttempts
	return l
}

func (l *LoggerWithContext) WithDuration(duration time.Duration) *LoggerWithContext {
	l.context.Duration = duration
	return l
}

// Structured logging methods that include context (DRY principle)
func (l *LoggerWithContext) Info(msg string, args ...any) {
	l.logWithContext(slog.LevelInfo, msg, args...)
}
func (l *LoggerWithContext) Debug(msg string, args ...any) {
	l.logWithContext(slog.LevelDebug, msg, args...)
}
func (l *LoggerWithContext) Warn(msg string, args ...any) {
	l.logWithContext(slog.LevelWarn, msg, args...)
}
func (l *LoggerWithContext) Error(msg string, args ...any) {
	l.logWithContext(slog.LevelError, msg, args...)
}

func (l *LoggerWithContext) logWithContext(level slog.Level, msg string, args ...any) {
	// Build context attributes
	contextArgs := make([]any, 0, 8+len(args))

	if l.context.InstanceID != "" {
		contextArgs = append(contextArgs, "instance_id", l.context.InstanceID)
	}
	if l.context.Operation != "" {
		contextArgs = append(contextArgs, "operation", l.context.Operation)
	}
	if l.context.Attempt > 0 {
		contextArgs = append(contextArgs, "attempt", l.context.Attempt)
		if l.context.MaxAttempts > 0 {
			contextArgs = append(contextArgs, "max_attempts", l.context.MaxAttempts)
		}
	}
	if l.context.Duration > 0 {
		contextArgs = append(contextArgs, "duration", l.context.Duration.String())
	}

	// Add original arguments
	contextArgs = append(contextArgs, args...)

	// Log with appropriate level
	l.Logger.Log(context.Background(), level, msg, contextArgs...)
}

// Handle é o método que formata cada entrada de log.
func (h *CustomTextHandler) Handle(ctx context.Context, r slog.Record) error {
	// Formata o nível do log com cor usando função centralizada.
	levelStr := colorizeLevel(r.Level, r.Level.String())

	// Monta a string de atributos (chave=valor) que vêm depois da mensagem.
	attrs := ""
	r.Attrs(func(a slog.Attr) bool {
		attrs = attrs + " " + color.CyanString(a.Key+"=") + a.Value.String()
		return true
	})

	// Monta a linha de log final no formato desejado.
	// Ex: 15:04:05.000 [INFO] Mensagem principal key=value
	logLine := fmt.Sprintf("%s [%s] %s%s\n",
		r.Time.Format("15:04:05.000"),
		levelStr,
		r.Message,
		attrs,
	)

	// Escreve a linha formatada na saída configurada.
	_, err := io.WriteString(h.output, logLine)
	return err
}

// WithAttrs e WithGroup são necessários para implementar a interface slog.Handler.
func (h *CustomTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CustomTextHandler{
		level:  h.level,
		output: h.output,
	}
}

func (h *CustomTextHandler) WithGroup(name string) slog.Handler {
	return &CustomTextHandler{
		level:  h.level,
		output: h.output,
	}
}

func (h *CustomTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}
