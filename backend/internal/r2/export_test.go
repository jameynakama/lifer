package r2

import "time"

// SetUploadRetryDelaysForTest overrides the retry ladder so tests that
// exhaust retries don't sleep through the production delays. Returns a
// restore func for t.Cleanup.
func SetUploadRetryDelaysForTest(delays []time.Duration) (restore func()) {
	orig := uploadRetryDelays
	uploadRetryDelays = delays
	return func() { uploadRetryDelays = orig }
}
