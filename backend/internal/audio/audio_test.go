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
