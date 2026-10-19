package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunCmd(t *testing.T) {
	t.Setenv("TO_KEEP", "parent")
	t.Setenv("TO_REMOVE", "old")

	outFile := filepath.Join(t.TempDir(), "out.txt")
	script := `printf '%s|%s|%s' "${TO_KEEP-p-missing}" "${TO_SET-s-missing}" "${TO_REMOVE-r-present}" > ` + outFile

	code := RunCmd([]string{"/bin/sh", "-c", script}, Environment{
		"TO_SET":    {Value: "new"},
		"TO_REMOVE": {NeedRemove: true},
	})
	if code != 0 {
		t.Fatalf("RunCmd: exit code = %d, want 0", code)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if got, want := string(data), "parent|new|r-present"; got != want {
		t.Errorf("child environment:\ngot  %q\nwant %q", got, want)
	}
}

func TestRunCmdExitCode(t *testing.T) {
	if code := RunCmd([]string{"/bin/sh", "-c", "exit 42"}, nil); code != 42 {
		t.Errorf("RunCmd: exit code = %d, want 42", code)
	}
}

func TestRunCmdBinaryNotFound(t *testing.T) {
	if code := RunCmd([]string{"/no/such/binary"}, nil); code == 0 {
		t.Error("RunCmd: expected non-zero exit code for missing binary, got 0")
	}
}

func TestRunCmdEmptyCmd(t *testing.T) {
	if code := RunCmd(nil, nil); code == 0 {
		t.Error("RunCmd: expected non-zero exit code for empty cmd, got 0")
	}
}
