// Package shared contains all shared code.
package shared

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

var (
	// Logger is the logger for the app.
	Logger zerolog.Logger
	done   bool
)

func deriveLoggingLevel() zerolog.Level {
	defaultLevel := zerolog.InfoLevel
	rawValue, exists := os.LookupEnv("fix_exif_date_logging_level")
	if !exists {
		return defaultLevel
	}

	value, found := map[string]zerolog.Level{
		"info":  zerolog.InfoLevel,
		"debug": zerolog.DebugLevel,
	}[strings.ToLower(rawValue)]
	if !found {
		return defaultLevel
	}
	return value
}

// InitLogging initializes logging.
func InitLogging() {
	if done {
		return
	}
	file, err := os.OpenFile("fix-exif-date.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		panic(err)
	}
	initLogger(file, deriveLoggingLevel())
}

// InitTestLogging creates a zerolog logger that writes to t.Log.
func InitTestLogging(tb testing.TB) {
	initLogger(&testWriter{tb: tb}, zerolog.DebugLevel)
}

func initLogger(writer io.Writer, level zerolog.Level) {
	Logger = zerolog.New(writer).Level(level).With().Timestamp().Logger()
	done = true
}

type testWriter struct {
	tb testing.TB
}

func (w testWriter) Write(p []byte) (n int, err error) {
	w.tb.Log(string(p))
	return len(p), nil
}
