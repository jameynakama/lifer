package main

import (
	"testing"

	"github.com/jameynakama/flockdeck/internal/audio"
	"github.com/stretchr/testify/assert"
)

func TestDecide(t *testing.T) {
	conformant := audio.Info{Format: "mp3", Channels: 1, BitRate: 96_000, Duration: 30}

	tests := []struct {
		name string
		info audio.Info
		want action
	}{
		{"uncompressed wav must be transcoded", audio.Info{Format: "wav", Channels: 2, BitRate: 1_536_000}, actionTranscode},
		{"stereo mp3 must be transcoded", audio.Info{Format: "mp3", Channels: 2, BitRate: 96_000}, actionTranscode},
		{"320k mono mp3 is over the ceiling", audio.Info{Format: "mp3", Channels: 1, BitRate: 320_000}, actionTranscode},
		{"flac must be transcoded", audio.Info{Format: "flac", Channels: 1, BitRate: 700_000}, actionTranscode},
		{"at the ceiling exactly is conformant", audio.Info{Format: "mp3", Channels: 1, BitRate: 112_000}, actionPeaksOnly},
		{"unknown bit rate is treated as unverified", audio.Info{Format: "mp3", Channels: 1, BitRate: 0}, actionTranscode},
		{"conformant baseline", conformant, actionPeaksOnly},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, decide(tt.info))
		})
	}
}
