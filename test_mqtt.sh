#!/bin/bash
# Quick MQTT upstream smoke test using mosquitto-clients only (binary protocol).
# For a full round-trip that also decodes the binary response, prefer:
#   go run ./cmd/devicesim -wav test.wav -broker tcp://localhost:1883
#
# Wire format (device -> server), device id in the topic:
#   /device/{id}/audio/start  JSON   {"kid_id":"..."}
#   /device/{id}/audio/chunk  binary [u16 index LE][u16 total LE][PCM]
#   /device/{id}/audio/end    empty
#
# Usage: ./test_mqtt.sh <test.wav> [broker] [port]

set -euo pipefail

if [ $# -lt 1 ]; then
  echo "Usage: $0 <test.wav> [broker] [port]"
  exit 1
fi

AUDIO_FILE=$1
DEVICE_ID="${DEVICE_ID:-test-device}"
BROKER="${2:-localhost}"
PORT="${3:-1883}"

[ -f "$AUDIO_FILE" ] || { echo "File $AUDIO_FILE not found"; exit 1; }
command -v mosquitto_pub >/dev/null || { echo "install mosquitto-clients"; exit 1; }

RAW=$(mktemp)
trap 'rm -f "$RAW" /tmp/cwlb_c_* /tmp/cwlb_p_*' EXIT

# strip 44-byte WAV header → raw PCM
dd if="$AUDIO_FILE" of="$RAW" bs=44 skip=1 2>/dev/null
echo "raw PCM: $(stat -c%s "$RAW") bytes"

# split into ~1s chunks
split -b 32000 --numeric-suffixes=0 "$RAW" /tmp/cwlb_c_
FILES=(/tmp/cwlb_c_*)
TOTAL=${#FILES[@]}
echo "chunks: $TOTAL"

mosquitto_pub -h "$BROKER" -p "$PORT" -q 1 \
  -t "/device/$DEVICE_ID/audio/start" -m "{\"kid_id\":\"KID001\"}"
echo "-> start"

# writes a little-endian uint16 header (index,total) + PCM to a payload file.
# Two-step printf avoids command substitution stripping 0x0a bytes from headers.
for i in "${!FILES[@]}"; do
  P="/tmp/cwlb_p_$i"
  HDR=$(printf '\\x%02x\\x%02x\\x%02x\\x%02x' \
    $((i & 255)) $(((i >> 8) & 255)) $((TOTAL & 255)) $(((TOTAL >> 8) & 255)))
  printf "$HDR" > "$P"
  cat "${FILES[$i]}" >> "$P"
  mosquitto_pub -h "$BROKER" -p "$PORT" -q 1 -t "/device/$DEVICE_ID/audio/chunk" -f "$P"
  echo "-> chunk $((i + 1))/$TOTAL ($(stat -c%s "${FILES[$i]}") bytes)"
done

mosquitto_pub -h "$BROKER" -p "$PORT" -q 1 -t "/device/$DEVICE_ID/audio/end" -m ""
echo "-> end sent"
echo ""
echo "Upstream sent. The server response is binary; decode it with:"
echo "  go run ./cmd/devicesim -wav $AUDIO_FILE -device $DEVICE_ID"
