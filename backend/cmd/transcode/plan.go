package main

import "github.com/jameynakama/flockdeck/internal/audio"

// bitRateCeiling is the highest bit rate an existing object may have and still
// count as already-transcoded. Slightly above the 96 kbps target so LAME's own
// variation does not send conformant files through a pointless re-encode.
const bitRateCeiling = 112_000

type action int

const (
	actionSkip action = iota
	actionPeaksOnly
	actionTranscode
)

func (a action) String() string {
	switch a {
	case actionSkip:
		return "skip"
	case actionPeaksOnly:
		return "peaks"
	case actionTranscode:
		return "transcode"
	default:
		return "unknown"
	}
}

// decide classifies a probed recording as needing a real re-encode or as
// already conformant. Callers skip rows that already have peaks before ever
// downloading them, so this is only reached for a recording whose peaks are
// missing -- and every such row is re-uploaded regardless of the answer here,
// since the transcode that measures the peaks is the same transcode that
// would produce the replacement object. decide only separates the report's
// counts: how much of the catalog needed a real re-encode versus was already
// conformant and only needed its object refreshed to match its new peaks.
//
// Keeping this pure is what makes the job idempotent and resumable: a run that
// uploads but fails its DB write comes back as actionPeaksOnly on the next pass
// and heals itself.
//
// A bit rate of 0 means ffprobe reported none, so the file cannot be confirmed
// conformant and must be re-encoded.
func decide(info audio.Info) action {
	conformant := info.Format == "mp3" &&
		info.Channels == 1 &&
		info.BitRate > 0 &&
		info.BitRate <= bitRateCeiling
	switch {
	case !conformant:
		return actionTranscode
	default:
		return actionPeaksOnly
	}
}
