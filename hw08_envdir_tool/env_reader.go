package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Environment map[string]EnvValue

// EnvValue helps to distinguish between empty files and files with the first empty line.
type EnvValue struct {
	Value      string
	NeedRemove bool
}

// ReadDir reads a specified directory and returns map of env variables.
// Variables represented as files where filename is name of variable, file first line is a value.
func ReadDir(dir string) (Environment, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}

	env := make(Environment, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		if strings.Contains(name, "=") {
			return nil, fmt.Errorf("invalid env variable name %q: must not contain '='", name)
		}

		filePath := filepath.Join(dir, name)

		info, err := os.Stat(filePath)
		if err != nil {
			return nil, fmt.Errorf("stat env file %q: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			continue
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read env file %q: %w", name, err)
		}

		if len(data) == 0 {
			env[name] = EnvValue{NeedRemove: true}
			continue
		}

		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			data = data[:i]
		}
		value := strings.TrimRight(string(data), " \t")
		value = strings.ReplaceAll(value, "\x00", "\n")
		env[name] = EnvValue{Value: value}
	}

	return env, nil
}
