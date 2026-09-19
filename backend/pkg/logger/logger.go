// Package logger builds the application logger on top of log/slog.
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Options configures a logger. Env "production" logs JSON, anything else text;
// Level is debug, info, warn or error. The zero value logs info to stderr.
type Options struct {
	Env    string
	Level  string
	Output io.Writer
}

// New builds a logger from opts.
func New(opts Options) *slog.Logger {
	out := opts.Output
	if out == nil {
		out = os.Stderr
	}

	handlerOpts := &slog.HandlerOptions{Level: ParseLevel(opts.Level)}

	var h slog.Handler
	if strings.EqualFold(opts.Env, "production") {
		h = slog.NewJSONHandler(out, handlerOpts)
	} else {
		h = slog.NewTextHandler(out, handlerOpts)
	}
	return slog.New(h)
}

// Init builds a logger and installs it as the slog default, so packages
// calling slog directly share this configuration.
func Init(opts Options) *slog.Logger {
	l := New(opts)
	slog.SetDefault(l)
	return l
}

// ParseLevel maps a level name to its slog value. Unknown names fall back
// to info rather than failing startup over a typo.
func ParseLevel(name string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Discard returns a logger that writes nowhere, for tests.
func Discard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
