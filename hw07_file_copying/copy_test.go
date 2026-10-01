package main

import (
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyOffsetEqualsFileSize(t *testing.T) {
	progressOut = io.Discard

	fromPath := filepath.Join("testdata", "input.txt")

	stat, err := os.Stat(fromPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	toPath := filepath.Join(t.TempDir(), "out.txt")
	if err := Copy(fromPath, toPath, stat.Size(), 0); err != nil {
		t.Fatalf("Copy with offset == file size: %v", err)
	}

	got, err := os.ReadFile(toPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d bytes, want 0", len(got))
	}
}

func TestCopyErrors(t *testing.T) {
	progressOut = io.Discard

	fromPath := filepath.Join("testdata", "input.txt")

	stat, err := os.Stat(fromPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	tests := []struct {
		name     string
		fromPath string
		offset   int64
		limit    int64
		wantErr  error
	}{
		{"offset exceeds file size", fromPath, stat.Size() + 1, 0, ErrOffsetExceedsFileSize},
		{"source not found", filepath.Join("testdata", "no_such_file.txt"), 0, 0, os.ErrNotExist},
		{"negative offset", fromPath, -1, 0, ErrInvalidArgs},
		{"negative limit", fromPath, 0, -1, ErrInvalidArgs},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toPath := filepath.Join(t.TempDir(), "out.txt")
			err := Copy(tt.fromPath, toPath, tt.offset, tt.limit)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got error %v, want %v", err, tt.wantErr)
			}
		})
	}

	t.Run("irregular file", func(t *testing.T) {
		if _, err := os.Stat(os.DevNull); err != nil {
			t.Skip("no", os.DevNull)
		}

		toPath := filepath.Join(t.TempDir(), "out.txt")
		if err := Copy(os.DevNull, toPath, 0, 0); !errors.Is(err, ErrUnsupportedFile) {
			t.Errorf("got error %v, want %v", err, ErrUnsupportedFile)
		}
	})

	t.Run("directory as source", func(t *testing.T) {
		toPath := filepath.Join(t.TempDir(), "out.txt")
		if err := Copy(t.TempDir(), toPath, 0, 0); !errors.Is(err, ErrUnsupportedFile) {
			t.Errorf("got error %v, want %v", err, ErrUnsupportedFile)
		}
	})

	t.Run("same file", func(t *testing.T) {
		samePath := filepath.Join(t.TempDir(), "same.txt")
		if err := os.WriteFile(samePath, []byte("data"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		if err := Copy(samePath, samePath, 0, 0); !errors.Is(err, ErrSameFile) {
			t.Errorf("got error %v, want %v", err, ErrSameFile)
		}
	})
}

func TestCopyEdgeCases(t *testing.T) {
	progressOut = io.Discard

	fromPath := filepath.Join("testdata", "input.txt")

	input, err := os.ReadFile(fromPath)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}

	tests := []struct {
		name   string
		offset int64
		limit  int64
		want   []byte
	}{
		{"limit not aligned to buffer", 0, 1025, input[:1025]},
		{"limit one byte", 0, 1, input[:1]},
		{"limit exactly one buffer", 0, bufSize, input[:bufSize]},
		{"limit max int64", 0, math.MaxInt64, input},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toPath := filepath.Join(t.TempDir(), "out.txt")
			if err := Copy(fromPath, toPath, tt.offset, tt.limit); err != nil {
				t.Fatalf("Copy(offset=%d, limit=%d): %v", tt.offset, tt.limit, err)
			}

			got, err := os.ReadFile(toPath)
			if err != nil {
				t.Fatalf("read output: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("got %d bytes, want %d bytes", len(got), len(tt.want))
			}
		})
	}

	t.Run("empty file", func(t *testing.T) {
		toPath := filepath.Join(t.TempDir(), "out.txt")
		if err := Copy(filepath.Join("testdata", "empty.txt"), toPath, 0, 0); err != nil {
			t.Fatalf("Copy: %v", err)
		}

		got, err := os.ReadFile(toPath)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}

		if len(got) != 0 {
			t.Errorf("got %d bytes, want 0", len(got))
		}
	})

	t.Run("overwrites existing destination", func(t *testing.T) {
		toPath := filepath.Join(t.TempDir(), "out.txt")
		if err := os.WriteFile(toPath, bytes.Repeat([]byte("x"), 4096), 0o644); err != nil {
			t.Fatalf("prepare destination: %v", err)
		}

		if err := Copy(fromPath, toPath, 0, 10); err != nil {
			t.Fatalf("Copy: %v", err)
		}

		got, err := os.ReadFile(toPath)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}

		if !bytes.Equal(got, input[:10]) {
			t.Errorf("got %d bytes, want %d bytes", len(got), 10)
		}
	})
}

func TestPercent(t *testing.T) {
	tests := []struct {
		copied, total int64
		want          int
	}{
		{0, 1000, 0},
		{1000, 1000, 100},
		{0, 0, 100},
		{1, 1, 100},
		{0, math.MaxInt64, 0},
		{500, 1000, 50},
	}
	for _, tt := range tests {
		if got := percent(tt.copied, tt.total); got != tt.want {
			t.Errorf("percent(%d, %d) = %d, want %d", tt.copied, tt.total, got, tt.want)
		}
	}

	prev := -1
	const total = 100_000
	for c := int64(0); c <= total; c += total / 1000 {
		p := percent(c, total)
		if p < prev {
			t.Fatalf("percent(%d, %d) = %d is not monotonic, previous was %d", c, total, p, prev)
		}
		prev = p
	}

	if prev != 100 {
		t.Errorf("last percent = %d, want 100", prev)
	}
}
