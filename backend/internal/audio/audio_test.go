package audio_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jameynakama/flockdeck/internal/audio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbe_ReportsStereoWAV(t *testing.T) {
	info, err := audio.Probe(context.Background(), "testdata/stereo.wav")
	require.NoError(t, err)

	assert.Equal(t, "wav", info.Format)
	assert.Equal(t, 2, info.Channels)
	assert.InDelta(t, 2.0, info.Duration, 0.1)
	// 44100 * 2ch * 16bit = 1411200 bps, uncompressed.
	assert.Greater(t, info.BitRate, 1_000_000)
}

func TestProbe_ReportsMonoMP3(t *testing.T) {
	info, err := audio.Probe(context.Background(), "testdata/mono320.mp3")
	require.NoError(t, err)

	assert.Equal(t, "mp3", info.Format)
	assert.Equal(t, 1, info.Channels)
	assert.InDelta(t, 320_000, info.BitRate, 20_000)
}

func TestProbe_MissingFileErrors(t *testing.T) {
	_, err := audio.Probe(context.Background(), "testdata/nope.wav")
	require.Error(t, err)
}

func TestGainFor_PullsDownHotFile(t *testing.T) {
	// hot.wav peaks at roughly 0 dBFS, so it must be attenuated to -1.
	gain, err := audio.GainForTest(context.Background(), "testdata/hot.wav")
	require.NoError(t, err)
	assert.InDelta(t, -1.0, gain, 0.5)
}

func TestGainFor_BoostsQuietFileUpToTheCap(t *testing.T) {
	// quiet.wav sits near -70 dBFS. Reaching -1 dBFS would need +69 dB, which
	// would just amplify noise, so the boost is capped.
	gain, err := audio.GainForTest(context.Background(), "testdata/quiet.wav")
	require.NoError(t, err)
	assert.Equal(t, audio.MaxBoostDB, gain)
}

func TestGainFor_LeavesModerateFileWithModerateBoost(t *testing.T) {
	// stereo.wav was built at -6 dBFS, so it needs about +5 dB.
	gain, err := audio.GainForTest(context.Background(), "testdata/stereo.wav")
	require.NoError(t, err)
	assert.InDelta(t, 5.0, gain, 1.0)
}

func TestTranscode_ProducesMonoMP3AtTargetBitrate(t *testing.T) {
	res, err := audio.Transcode(context.Background(), "testdata/stereo.wav")
	require.NoError(t, err)
	require.NotEmpty(t, res.Data)

	out := writeTemp(t, res.Data)
	info, err := audio.Probe(context.Background(), out)
	require.NoError(t, err)

	assert.Equal(t, "mp3", info.Format)
	assert.Equal(t, 1, info.Channels)
	assert.InDelta(t, 96_000, info.BitRate, 15_000)
	assert.InDelta(t, 2.0, info.Duration, 0.2)
	assert.Less(t, len(res.Data), 40_000, "2 s at 96 kbps should be well under 40 KB")
}

func TestTranscode_NormalizesPeakToTarget(t *testing.T) {
	res, err := audio.Transcode(context.Background(), "testdata/stereo.wav")
	require.NoError(t, err)

	out := writeTemp(t, res.Data)
	gain, err := audio.GainForTest(context.Background(), out)
	require.NoError(t, err)
	// The output already peaks at -1 dBFS, so it needs no further adjustment.
	assert.InDelta(t, 0.0, gain, 1.0)
}

func TestTranscode_ReturnsPeaksInRange(t *testing.T) {
	res, err := audio.Transcode(context.Background(), "testdata/stereo.wav")
	require.NoError(t, err)

	require.Len(t, res.Peaks, audio.PeakCount)
	var sawNonZero bool
	for i, p := range res.Peaks {
		require.GreaterOrEqual(t, p, int16(0), "peak %d below range", i)
		require.LessOrEqual(t, p, int16(255), "peak %d above range", i)
		if p > 0 {
			sawNonZero = true
		}
	}
	assert.True(t, sawNonZero, "a tone should produce non-zero peaks")
}

func TestTranscode_NearSilentFileIsNotBoostedToFullScale(t *testing.T) {
	res, err := audio.Transcode(context.Background(), "testdata/quiet.wav")
	require.NoError(t, err)

	// Capped at +20 dB from about -70 dBFS, so the output stays far below
	// full scale and its peaks stay small.
	var maxPeak int16
	for _, p := range res.Peaks {
		if p > maxPeak {
			maxPeak = p
		}
	}
	assert.Less(t, maxPeak, int16(64), "capped boost must not reach full scale")
}

func TestTranscode_MissingFileErrors(t *testing.T) {
	_, err := audio.Transcode(context.Background(), "testdata/nope.wav")
	require.Error(t, err)
}

func writeTemp(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "out.mp3")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path
}
