package audio_test

import (
	"context"
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
