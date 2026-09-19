package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/financeapp/backend/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"DEBUG":   slog.LevelDebug,
		" warn ":  slog.LevelWarn,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
		"info":    slog.LevelInfo,
		"":        slog.LevelInfo,
		"verbose": slog.LevelInfo, // a typo must not change the level
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, want, logger.ParseLevel(name))
		})
	}
}

func TestNew_ProductionLogsJSON(t *testing.T) {
	var buf bytes.Buffer
	logger.New(logger.Options{Env: "production", Output: &buf}).
		Info("server listening", "addr", ":8080")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry), "production output must be JSON")
	assert.Equal(t, "server listening", entry["msg"])
	assert.Equal(t, ":8080", entry["addr"])
	assert.Equal(t, "INFO", entry["level"])
}

func TestNew_DevelopmentLogsText(t *testing.T) {
	var buf bytes.Buffer
	logger.New(logger.Options{Env: "development", Output: &buf}).Info("hello", "k", "v")

	out := buf.String()
	assert.Contains(t, out, "msg=hello")
	assert.Contains(t, out, "k=v")
	assert.NotContains(t, out, `"msg"`, "development output must not be JSON")
}

func TestNew_LevelFiltersQuieterRecords(t *testing.T) {
	var buf bytes.Buffer
	l := logger.New(logger.Options{Level: "warn", Output: &buf})

	l.Debug("debug line")
	l.Info("info line")
	assert.Empty(t, buf.String(), "records below the level must be dropped")

	l.Warn("warn line")
	assert.Contains(t, buf.String(), "warn line")
}

func TestNew_DefaultsToStderrWithoutPanicking(t *testing.T) {
	assert.NotPanics(t, func() { logger.New(logger.Options{}) },
		"a zero Options must still produce a usable logger")
}

func TestInit_SetsTheSlogDefault(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })

	var buf bytes.Buffer
	logger.Init(logger.Options{Env: "production", Output: &buf})

	// Packages that log through slog directly must pick this up.
	slog.Info("from the package default")
	assert.Contains(t, buf.String(), "from the package default")
}
