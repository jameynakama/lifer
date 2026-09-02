// Package audio wraps the ffmpeg and ffprobe CLIs. It is the only place in the
// codebase that shells out to them, so the encode settings used at ingest and
// during backfill can never drift apart.
package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

const (
	// TargetPeakDBFS is where the loudest sample lands after normalization.
	// Peak normalization is a linear gain, so signal-to-noise is preserved
	// exactly -- unlike loudnorm, which targets integrated loudness and would
	// apply far more gain to the mostly-quiet recordings xeno-canto carries,
	// lifting hiss and insects to audible levels.
	TargetPeakDBFS = -1.0

	// MaxBoostDB caps upward gain. A file needing more than this is effectively
	// silent; boosting it would raise noise to full scale for no benefit.
	MaxBoostDB = 20.0
)

var maxVolumeRe = regexp.MustCompile(`max_volume:\s*(-?[0-9.]+) dB`)

// gainFor measures the file's peak and returns the dB adjustment that puts it
// at TargetPeakDBFS, clamped to MaxBoostDB.
func gainFor(ctx context.Context, path string) (float64, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", path, "-af", "volumedetect", "-f", "null", "-")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var execErr *exec.Error
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return 0, ErrFFmpegMissing
		}
		return 0, fmt.Errorf("volumedetect %s: %w: %s", path, err, stderr.String())
	}

	m := maxVolumeRe.FindSubmatch(stderr.Bytes())
	if m == nil {
		return 0, fmt.Errorf("volumedetect %s: no max_volume in output", path)
	}
	maxVolume, err := strconv.ParseFloat(string(m[1]), 64)
	if err != nil {
		return 0, fmt.Errorf("volumedetect %s: parse max_volume %q: %w", path, m[1], err)
	}

	return min(TargetPeakDBFS-maxVolume, MaxBoostDB), nil
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

// PeakCount is how many waveform buckets Transcode emits, regardless of the
// recording's length. WavePlayer draws a fixed number of bars, so a fixed
// bucket count keeps the payload small and the drawing code trivial.
const PeakCount = 1000

// peaksSampleRate is the rate the peaks stream is decoded at. The waveform only
// needs an envelope, so 8 kHz keeps the byte stream small: even a 180 s
// recording is under 3 MB.
const peaksSampleRate = 8000

// Result is one decode's two artifacts.
type Result struct {
	Data  []byte  // the encoded mp3
	Peaks []int16 // PeakCount buckets, each 0..255
}

// Transcode re-encodes the file at path to mono 96 kbps mp3, peak-normalized to
// TargetPeakDBFS, and returns the encoded bytes alongside a PeakCount-bucket
// waveform envelope.
//
// It runs ffmpeg twice: once to measure the peak (a pipe cannot rewind, which
// is why this takes a path rather than an io.Reader), then once to encode. The
// encode pass writes both outputs from a single decode -- the mp3 to a temp
// file and a low-rate PCM stream to stdout -- so the peaks describe exactly the
// audio that ends up playing, gain included.
func Transcode(ctx context.Context, path string) (Result, error) {
	gain, err := gainFor(ctx, path)
	if err != nil {
		return Result{}, err
	}

	tmpDir, err := os.MkdirTemp("", "flockdeck-transcode-")
	if err != nil {
		return Result{}, fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck
	outPath := filepath.Join(tmpDir, "out.mp3")

	gainArg := "volume=" + strconv.FormatFloat(gain, 'f', 2, 64) + "dB"
	pcm, err := run(ctx, "ffmpeg",
		"-v", "error",
		"-i", path,
		// Output 1: the encoded file.
		"-af", gainArg, "-ac", "1", "-ar", "44100",
		"-c:a", "libmp3lame", "-b:a", "96k", "-f", "mp3", outPath,
		// Output 2: raw PCM for the waveform. -af is a per-output option, so
		// both outputs come from the same decode.
		"-af", gainArg, "-ac", "1", "-ar", strconv.Itoa(peaksSampleRate),
		"-c:a", "pcm_s16le", "-f", "s16le", "-",
	)
	if err != nil {
		return Result{}, err
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		return Result{}, fmt.Errorf("read encoded output: %w", err)
	}

	return Result{Data: data, Peaks: bucketPeaks(pcm)}, nil
}

// bucketPeaks reduces a little-endian signed 16-bit mono PCM stream to
// PeakCount buckets, each the loudest absolute sample in its slice of the
// stream scaled to 0..255.
func bucketPeaks(pcm []byte) []int16 {
	peaks := make([]int16, PeakCount)
	samples := len(pcm) / 2
	if samples == 0 {
		return peaks
	}

	for i := range PeakCount {
		start := i * samples / PeakCount
		end := (i + 1) * samples / PeakCount
		if end <= start {
			end = start + 1
		}
		if end > samples {
			end = samples
		}

		var loudest int32
		for s := start; s < end; s++ {
			v := int32(int16(binary.LittleEndian.Uint16(pcm[s*2:])))
			if v < 0 {
				v = -v
			}
			if v > loudest {
				loudest = v
			}
		}
		peaks[i] = int16(loudest * 255 / 32768)
	}
	return peaks
}
