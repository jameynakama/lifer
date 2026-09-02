// Package audio wraps the ffmpeg and ffprobe CLIs. It is the only place in the
// codebase that shells out to them, so the encode settings used at ingest and
// during backfill can never drift apart.
package audio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ErrFFmpegMissing is returned when ffmpeg or ffprobe is not on PATH. Ingest and
// the transcode job both run from a laptop, where that is a plausible mistake.
var ErrFFmpegMissing = errors.New("ffmpeg and ffprobe must be installed and on PATH")

// Info is what ffprobe reports about a source file. BitRate is bits per second
// and is 0 when ffprobe reports none (some containers omit it).
type Info struct {
	Format   string
	Channels int
	BitRate  int
	Duration float64
}

// Probe reports the container format, channel count, bit rate, and duration of
// the file at path. The extension is never trusted: the bucket holds WAV and
// FLAC data under .mp3 keys, so callers must probe.
func Probe(ctx context.Context, path string) (Info, error) {
	out, err := run(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=format_name,duration,bit_rate",
		"-show_entries", "stream=channels",
		"-select_streams", "a:0",
		"-of", "json",
		path,
	)
	if err != nil {
		return Info{}, err
	}

	var probed struct {
		Format struct {
			FormatName string `json:"format_name"`
			Duration   string `json:"duration"`
			BitRate    string `json:"bit_rate"`
		} `json:"format"`
		Streams []struct {
			Channels int `json:"channels"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &probed); err != nil {
		return Info{}, fmt.Errorf("parse ffprobe output for %s: %w", path, err)
	}
	if len(probed.Streams) == 0 {
		return Info{}, fmt.Errorf("no audio stream in %s", path)
	}

	info := Info{
		Format:   normalizeFormat(probed.Format.FormatName),
		Channels: probed.Streams[0].Channels,
	}
	// Both fields are optional in ffprobe output; a parse failure means absent,
	// not malformed, so the zero value is the right answer.
	info.Duration, _ = strconv.ParseFloat(probed.Format.Duration, 64)
	info.BitRate, _ = strconv.Atoi(probed.Format.BitRate)
	return info, nil
}

// normalizeFormat collapses ffprobe's comma-separated format lists ("mov,mp4,
// m4a,3gp,3g2,mj2") and its WAV alias to a single stable token.
func normalizeFormat(name string) string {
	for _, part := range strings.Split(name, ",") {
		switch part {
		case "wav", "mp3", "flac", "ogg", "mp4":
			return part
		}
	}
	if name == "" {
		return ""
	}
	return strings.Split(name, ",")[0]
}

// run executes an ffmpeg-family command and returns its stdout. Stderr is
// folded into the error because ffmpeg reports failures there.
func run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		var execErr *exec.Error
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return nil, ErrFFmpegMissing
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("%s: %w: %s", name, err, exitErr.Stderr)
		}
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}
