package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

var errFFmpegNotFound = errors.New("ffmpeg not found")

// ffmpegPath is the ffmpeg binary used for merging, resolved by findFFmpeg at
// startup.
var ffmpegPath = "ffmpeg"

// findFFmpeg locates ffmpeg on PATH, in the current directory, or next to the
// downloader's own executable.
func findFFmpeg() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		exe = ""
	}
	return findFFmpegFrom(exe)
}

// findFFmpegFrom is findFFmpeg with the downloader's executable path passed in.
//
// Windows users usually drop ffmpeg.exe into the same folder as the
// downloader. Windows searches the current directory implicitly, but since
// Go 1.19 exec refuses a binary found that way (exec.ErrDot), which made the
// merge fail after every track had already been downloaded. Placing ffmpeg
// there is deliberate, so it's accepted as an absolute path.
func findFFmpegFrom(exe string) (string, error) {
	path, err := exec.LookPath("ffmpeg")
	if err == nil {
		return path, nil
	}
	if errors.Is(err, exec.ErrDot) {
		return filepath.Abs(path)
	}

	if exe != "" {
		if path, err := exec.LookPath(filepath.Join(filepath.Dir(exe), "ffmpeg")); err == nil {
			return path, nil
		}
	}
	return "", errFFmpegNotFound
}
