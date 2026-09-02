#!/usr/bin/env bash
# Regenerates the audio test fixtures. Run from this directory.
# Committed outputs are small (<100 KB total); regenerate only if the
# expectations in audio_test.go change.
set -euo pipefail
cd "$(dirname "$0")"

# 2 s stereo 44.1 kHz WAV, two tones, peaking around -6 dBFS.
ffmpeg -y -f lavfi -i "sine=frequency=2000:duration=2:sample_rate=44100" \
       -f lavfi -i "sine=frequency=5000:duration=2:sample_rate=44100" \
       -filter_complex "[0:a][1:a]amerge=inputs=2,volume=-6dB[a]" \
       -map "[a]" -c:a pcm_s16le stereo.wav

# 2 s mono 320 kbps MP3 -- stands in for the already-compressed rows.
ffmpeg -y -f lavfi -i "sine=frequency=3000:duration=2:sample_rate=44100" \
       -ac 1 -c:a libmp3lame -b:a 320k mono320.mp3

# 2 s mono WAV pushed to full scale -- the "blaring Barn Owl" case.
ffmpeg -y -f lavfi -i "sine=frequency=1000:duration=2:sample_rate=44100" \
       -ac 1 -af "volume=0dB" -c:a pcm_s16le hot.wav

# 2 s mono WAV at roughly -70 dBFS -- below the +20 dB boost cap.
ffmpeg -y -f lavfi -i "sine=frequency=1000:duration=2:sample_rate=44100" \
       -ac 1 -af "volume=-70dB" -c:a pcm_s16le quiet.wav
