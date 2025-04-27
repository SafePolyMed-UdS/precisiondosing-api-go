package logger

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"precisiondosing-api-go/cfg"
	"time"

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

func LogServerError(c *gin.Context, err error) {
	log.Error().
		Str("endpoint", c.FullPath()).
		Str("method", c.Request.Method).
		Int("status", http.StatusInternalServerError).
		Err(err).
		Msg("Internal server error")
}

func LogForbiddenError(c *gin.Context, msg string) {
	log.Info().
		Str("endpoint", c.FullPath()).
		Str("method", c.Request.Method).
		Str("ip", c.ClientIP()).
		Str("user_agent", c.GetHeader("User-Agent")).
		Int("status", http.StatusUnauthorized).
		Msg(msg)
}

func LogUnauthorizedError(c *gin.Context) {
	log.Info().
		Str("endpoint", c.FullPath()).
		Str("method", c.Request.Method).
		Str("ip", c.ClientIP()).
		Str("user_agent", c.GetHeader("User-Agent")).
		Int("status", http.StatusUnauthorized).
		Msg("Unauthorized")
}

func LogInfo(msg string, fields map[string]interface{}) {
	event := log.Info()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func LogDebug(msg string, fields map[string]interface{}) {
	event := log.Debug()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func LogError(err error, msg string, fields map[string]interface{}) {
	event := log.Error().Err(err)
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func writer(debug bool, json bool) io.Writer {
	if !json {
		return zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, NoColor: debug}
	}
	return os.Stdout
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
