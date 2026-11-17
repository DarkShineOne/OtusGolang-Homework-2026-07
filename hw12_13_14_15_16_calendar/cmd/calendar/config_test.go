package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	path := writeConfig(t, `
logger:
  level: debug
  file: logs/test.log
http:
  host: 127.0.0.1
  port: 8888
storage:
  type: sql
  dsn: postgres://local:local@localhost:5432/calendar?sslmode=disable
`)

	config, err := NewConfig(path)
	require.NoError(t, err)
	require.Equal(t, "debug", config.Logger.Level)
	require.Equal(t, 8888, config.HTTP.Port)
	require.Equal(t, "sql", config.Storage.Type)
}

func TestNewConfigValidation(t *testing.T) {
	tests := map[string]string{
		"invalid yaml":    "logger: [",
		"invalid level":   validConfig("verbose", "memory", ""),
		"invalid storage": validConfig("info", "file", ""),
		"missing dsn":     validConfig("info", "sql", ""),
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewConfig(writeConfig(t, content))
			require.Error(t, err)
		})
	}
	_, err := NewConfig(filepath.Join(t.TempDir(), "missing.yaml"))
	require.Error(t, err)
}

func validConfig(level, storageType, dsn string) string {
	return "logger:\n  level: " + level + "\n  file: logs/test.log\n" +
		"http:\n  host: 127.0.0.1\n  port: 8080\n" +
		"storage:\n  type: " + storageType + "\n  dsn: " + dsn + "\n"
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}
