#!/bin/bash

# ./test_mqtt.sh test.wav [broker] [port]

if [ $# -lt 1 ]; then
  echo "Usage: $0 <test.wav> [broker] [port]"
  exit 1
fi

AUDIO_FILE=$1
DEVICE_ID="test-device"
BROKER=${2:-"localhost"}
PORT=${3:-"1883"}

# check if file exists
if [ ! -f "$AUDIO_FILE" ]; then
  echo "File $AUDIO_FILE not found"
  exit 1
fi

echo "Testing MQTT audio processing with $AUDIO_FILE on $BROKER:$PORT"

# check if mosquitto_pub is available
if ! command -v mosquitto_pub &>/dev/null; then
  echo "mosquitto_pub not found. Install mosquitto-clients."
  exit 1
fi

# convert WAV to raw PCM (skip header)
RAW_FILE="${AUDIO_FILE%.wav}.raw"
echo "Converting WAV to raw PCM..."
dd if="$AUDIO_FILE" of="$RAW_FILE" bs=44 skip=1 2>/dev/null
if [ ! -f "$RAW_FILE" ]; then
  echo "Failed to create raw file"
  exit 1
fi

RAW_SIZE=$(stat -c%s "$RAW_FILE")
echo "RAW_FILE size: $RAW_SIZE bytes"

# split into chunks of 32000 bytes (~1 sec at 16kHz 16-bit mono)
CHUNK_SIZE=32000
echo "Splitting into chunks..."
split -b $CHUNK_SIZE "$RAW_FILE" chunk_

# count chunks
CHUNK_COUNT=$(ls chunk_* 2>/dev/null | wc -l)
echo "Created $CHUNK_COUNT chunks: $(ls chunk_* 2>/dev/null)"

# send start
echo "Sending start message..."
START_MSG='{"robot_id": "'$DEVICE_ID'", "kid_id": "KID001"}'
mosquitto_pub -h $BROKER -p $PORT -t "/device/audio/start" -m "$START_MSG" -d

# send chunks
CHUNK_INDEX=0
for chunk in chunk_*; do
  if [ -f "$chunk" ]; then
    # base64 encode
    ENCODED=$(base64 -w 0 "$chunk")
    echo "Sending chunk $CHUNK_INDEX (size: $(stat -c%s "$chunk") bytes)..."
    mosquitto_pub -h $BROKER -p $PORT -t "/device/audio/chunk" -m "$ENCODED" -d
    CHUNK_INDEX=$((CHUNK_INDEX + 1))
  fi
done

# send end
echo "Sending end message..."
mosquitto_pub -h $BROKER -p $PORT -t "/device/audio/end" -m "done" -d

# clean up
rm -f "$RAW_FILE" chunk_*

echo "All messages sent. Starting subscription to response topics..."

# collect response chunks
CHUNK_COUNT=0
TOTAL_CHUNKS=0
OUTPUT_DIR="tmp-esp32-output"

# clean previous
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# subscribe to response chunks
TEMP_SUB="temp_sub.txt"
rm -f "$TEMP_SUB"
mosquitto_sub -h $BROKER -p $PORT -t "/device/$DEVICE_ID/audio/response_chunk" -t "/device/$DEVICE_ID/audio/response_end" >"$TEMP_SUB" &
SUB_PID=$!
sleep 1 # wait for sub to start

# wait a bit for messages
sleep 15
kill $SUB_PID 2>/dev/null

# Process the received lines
echo "Processing received messages from $TEMP_SUB:"
cat "$TEMP_SUB"
echo ""
while read -r line; do
  printf "Processing line: %q\n" "$line"
  # if it's response_end, process
  if [[ "$line" == *"response_end"* ]]; then
    echo "Response complete."
    break
  fi
  # parse json: {"index": 0, "total": 8, "data": "base64..."}
  # extract data using sed
  DATA=$(echo "$line" | sed 's/.*"data":\s*"\([^"]*\)".*/\1/' 2>/dev/null)
  INDEX=$(echo "$line" | sed 's/.*"index":\s*\([0-9]*\).*/\1/' 2>/dev/null)
  TOTAL=$(echo "$line" | sed 's/.*"total":\s*\([0-9]*\).*/\1/' 2>/dev/null)
  printf "Extracted INDEX: '%s', TOTAL: '%s', DATA length: %d\n" "$INDEX" "$TOTAL" "${#DATA}"
  # if no data from json, assume line is base64
  if [ -z "$DATA" ] && [[ "$line" != *"response_end"* ]]; then
    DATA="$line"
    INDEX="$CHUNK_COUNT"
    TOTAL=1 # single chunk if not json
  fi
  if [ -n "$DATA" ]; then
    printf "DATA preview: %.50s...\n" "$DATA"
    CHUNK_FILE="$OUTPUT_DIR/chunk_$(printf "%04d" $INDEX).pcm"
    if echo "$DATA" | base64 -d >"$CHUNK_FILE" 2>/dev/null; then
      SIZE=$(stat -c%s "$CHUNK_FILE" 2>/dev/null || echo 0)
      echo "Decoded and saved chunk $INDEX to $CHUNK_FILE (size: $SIZE bytes), total collected: $((CHUNK_COUNT + 1))"
      CHUNK_COUNT=$((CHUNK_COUNT + 1))
      TOTAL_CHUNKS=$TOTAL
    else
      echo "Failed to decode DATA: '$DATA'"
    fi
  fi
done <"$TEMP_SUB"

rm -f "$TEMP_SUB"

# check if directory has all chunks
if [ -d "$OUTPUT_DIR" ] && [ "$CHUNK_COUNT" -eq "$TOTAL_CHUNKS" ]; then
  echo "Saved $CHUNK_COUNT chunks to $OUTPUT_DIR/"
  echo "To combine into PCM: cat $OUTPUT_DIR/chunk_*.pcm > combined.pcm"
  echo "To convert to WAV: sox -r 16000 -b 16 -c 1 -e signed-integer combined.pcm esp32Output.wav"
  echo "Or play: aplay -r 16000 -f S16_LE combined.pcm"
  echo "ESP32 would now play this audio."
  cat ./tmp-esp32-output/chunk_*.pcm >combined.pcm
  aplay -r 16000 -f S16_LE combined.pcm
else
  echo "Incomplete response: received $CHUNK_COUNT of $TOTAL_CHUNKS chunks"
fi

echo "Test complete. If no response, check backend logs and broker connection."
