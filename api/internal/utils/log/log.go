package log

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"precisiondosing-api-go/cfg"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func MustInit(logConfig cfg.LogConfig, debug bool) {
	zerLogLevel, err := logLevel(logConfig.Level, debug)
	if err != nil {
		panic(fmt.Sprintf("Error setting log level: %v", err))
	}

	writer := writer(debug, logConfig.JSONFormat)
	baseLogger := zerolog.New(writer).With().Timestamp()

	if logConfig.LogCaller {
		baseLogger = baseLogger.Caller()
	}

	log.Logger = baseLogger.Logger()
	zerolog.SetGlobalLevel(zerLogLevel)
}

func writer(debug bool, useJSON bool) io.Writer {
	if useJSON {
		return os.Stdout
	}

	return &zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
		NoColor:    !debug,
		FormatLevel: func(i any) string {
			level := strings.ToUpper(fmt.Sprintf("%s", i))
			switch level {
			case "DEBUG":
				return "| \033[36mDBG\033[0m |"
			case "INFO":
				return "| \033[32mINF\033[0m |"
			case "WARN":
				return "| \033[33mWRN\033[0m |"
			case "ERROR":
				return "| \033[31mERR\033[0m |"
			case "FATAL":
				return "| \033[35mFTL\033[0m |"
			default:
				return level
			}
		},
	}
}

func logLevel(logLevel string, debug bool) (zerolog.Level, error) {
	zerLogLevel, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		return zerolog.DebugLevel, fmt.Errorf("cannot parse log level: %w", err)
	}
	if debug {
		zerLogLevel = zerolog.DebugLevel
	}
	return zerLogLevel, nil
}

type Logger struct {
	l zerolog.Logger
}

// Helper
func WithComponent(component string) Logger {
	return Logger{
		l: log.With().Str("component", component).Logger(),
	}
}

func (lg Logger) Error(msg string, fields ...func(e *zerolog.Event)) {
	event := lg.l.Error()
	for _, f := range fields {
		f(event)
	}
	event.Msg(msg)
}

func (lg Logger) Warn(msg string, fields ...func(e *zerolog.Event)) {
	event := lg.l.Warn()
	for _, f := range fields {
		f(event)
	}
	event.Msg(msg)
}

func (lg Logger) Info(msg string, fields ...func(e *zerolog.Event)) {
	event := lg.l.Info()
	for _, f := range fields {
		f(event)
	}
	event.Msg(msg)
}

func (lg Logger) Debug(msg string, fields ...func(e *zerolog.Event)) {
	event := lg.l.Debug()
	for _, f := range fields {
		f(event)
	}
	event.Msg(msg)
}

func (lg Logger) Panic(msg string, fields ...func(e *zerolog.Event)) {
	event := lg.l.Error()
	for _, f := range fields {
		f(event)
	}
	event.Msg(msg)
	panic(msg)
}

func Str(key, val string) func(e *zerolog.Event) {
	return func(e *zerolog.Event) {
		e.Str(key, val)
	}
}

func Strs(key string, val []string) func(e *zerolog.Event) {
	return func(e *zerolog.Event) {
		e.Strs(key, val)
	}
}

func Int(key string, val int) func(e *zerolog.Event) {
	return func(e *zerolog.Event) {
		e.Int(key, val)
	}
}

func Bool(key string, val bool) func(e *zerolog.Event) {
	return func(e *zerolog.Event) {
		e.Bool(key, val)
	}
}

func Err(err error) func(e *zerolog.Event) {
	return func(e *zerolog.Event) {
		e.Err(err)
	}
}

// API Logger
func ServerError(c *gin.Context, err error) {
	log.Error().
		Str("endpoint", c.FullPath()).
		Str("method", c.Request.Method).
		Int("status", http.StatusInternalServerError).
		Err(err).
		Msg("Internal server error")
}

func Forbidden(c *gin.Context, msg string) {
	log.Warn().
		Str("endpoint", c.FullPath()).
		Str("method", c.Request.Method).
		Str("ip", c.ClientIP()).
		Str("user_agent", c.GetHeader("User-Agent")).
		Int("status", http.StatusForbidden).
		Msg(msg)
}

func Unauthorized(c *gin.Context) {
	log.Warn().
		Str("endpoint", c.FullPath()).
		Str("method", c.Request.Method).
		Str("ip", c.ClientIP()).
		Str("user_agent", c.GetHeader("User-Agent")).
		Int("status", http.StatusUnauthorized).
		Msg("Unauthorized")
}

func BadRequest(c *gin.Context, msg string) {
	log.Warn().
		Str("endpoint", c.FullPath()).
		Str("method", c.Request.Method).
		Str("ip", c.ClientIP()).
		Str("user_agent", c.GetHeader("User-Agent")).
		Int("status", http.StatusBadRequest).
		Msg(msg)
}
