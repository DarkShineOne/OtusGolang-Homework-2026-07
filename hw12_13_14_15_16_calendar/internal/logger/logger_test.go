package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoggerFiltersLevels(t *testing.T) {
	var output bytes.Buffer
	logg, err := New("warn", &output)
	require.NoError(t, err)

	logg.Debug("debug message")
	logg.Info("info message")
	logg.Warn("warn message")
	logg.Error("error message")

	result := output.String()
	require.NotContains(t, result, "debug message")
	require.NotContains(t, result, "info message")
	require.Contains(t, result, "WARN")
	require.Contains(t, result, "ERROR")
}

func TestLoggerDebugLevel(t *testing.T) {
	var output bytes.Buffer
	logg, err := New("DEBUG", &output)
	require.NoError(t, err)
	logg.Debug("visible")
	require.Contains(t, output.String(), "DEBUG visible")
}

func TestLoggerInvalidLevel(t *testing.T) {
	_, err := New("verbose", &bytes.Buffer{})
	require.Error(t, err)
}
