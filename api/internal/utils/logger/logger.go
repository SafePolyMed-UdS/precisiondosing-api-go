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

	writer := writer(debug)
	zerolog.SetGlobalLevel(zerLogLevel)
	log.Logger = zerolog.New(writer).With().Timestamp().Logger()
}

func LogServerError(c *gin.Context, err error) {
	log.Error().
		Str("endpoint", c.FullPath()).
		Str("method", c.Request.Method).
		Int("status", http.StatusInternalServerError).
		Err(err).
		Msg("Internal server error")
}

func LogInternalError(err error) {
	log.Error().
		Err(err).
		Msg("Internal error")
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

func writer(debug bool) io.Writer {
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, NoColor: !debug}
	return consoleWriter
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
