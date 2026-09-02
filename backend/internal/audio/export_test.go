package audio

import "context"

// GainForTest exposes gainFor to the external test package. gainFor stays
// unexported because callers only ever want Transcode, which applies the gain
// itself; the test needs to pin the clamping behavior directly.
func GainForTest(ctx context.Context, path string) (float64, error) {
	return gainFor(ctx, path)
}
