package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeFakeFFmpeg creates an executable named like ffmpeg in dir and returns
// its path.
func writeFakeFFmpeg(t *testing.T, dir string) string {
	t.Helper()
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertSameFile(t *testing.T, got, want string) {
	t.Helper()
	if !filepath.IsAbs(got) {
		t.Errorf("got relative path %q, want an absolute one", got)
	}
	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat %q: %v", got, err)
	}
	wantInfo, err := os.Stat(want)
	if err != nil {
		t.Fatalf("stat %q: %v", want, err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFindFFmpegOnPath(t *testing.T) {
	dir := t.TempDir()
	want := writeFakeFFmpeg(t, dir)
	t.Setenv("PATH", dir)

	got, err := findFFmpegFrom("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSameFile(t, got, want)
}

func TestFindFFmpegInCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	want := writeFakeFFmpeg(t, dir)
	t.Chdir(dir)
	// A relative PATH entry makes exec.LookPath return exec.ErrDot on every
	// OS, reproducing what Windows does for the current directory.
	t.Setenv("PATH", ".")

	got, err := findFFmpegFrom("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSameFile(t, got, want)
}

func TestFindFFmpegNextToExecutable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	exeDir := t.TempDir()
	want := writeFakeFFmpeg(t, exeDir)

	got, err := findFFmpegFrom(filepath.Join(exeDir, "crunchyroll-downloader"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSameFile(t, got, want)
}

func TestFindFFmpegMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	exeDir := t.TempDir()

	_, err := findFFmpegFrom(filepath.Join(exeDir, "crunchyroll-downloader"))
	if !errors.Is(err, errFFmpegNotFound) {
		t.Errorf("got error %v, want %v", err, errFFmpegNotFound)
	}
}
