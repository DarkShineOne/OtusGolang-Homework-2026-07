package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadDir(t *testing.T) {
	env, err := ReadDir(filepath.Join("testdata", "env"))
	if err != nil {
		t.Fatalf("ReadDir: unexpected error: %v", err)
	}

	want := Environment{
		"FOO":   {Value: "   foo\nwith new line"},
		"BAR":   {Value: "bar"},
		"HELLO": {Value: `"hello"`},
		"UNSET": {NeedRemove: true},
		"EMPTY": {Value: ""},
	}
	if !reflect.DeepEqual(env, want) {
		t.Errorf("ReadDir = %#v, want %#v", env, want)
	}
}

func TestReadDirNameWithEquals(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "NA=ME"), []byte("value"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := ReadDir(dir); err == nil {
		t.Error("ReadDir: expected error for name containing '=', got nil")
	}
}

func TestReadDirMissingDir(t *testing.T) {
	if _, err := ReadDir(filepath.Join("testdata", "no-such-dir")); err == nil {
		t.Error("ReadDir: expected error for missing dir, got nil")
	}
}

func TestReadDirSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("linked\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "LINKED")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	env, err := ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: unexpected error: %v", err)
	}
	if got, want := env["LINKED"], (EnvValue{Value: "linked"}); got != want {
		t.Errorf("ReadDir() LINKED = %#v, want %#v", got, want)
	}
}
