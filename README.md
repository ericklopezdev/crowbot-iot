# Crowbot - IOT Assistant Project

## Project Overview

> Modern days are turning gray because of the overconsumption of social media content and dependency on AI tools. Especially for the next generations.

**Crowbot** emerges as a counterpoint to this trend - an IoT voice assistant that reimagines AI as a positive force for children's development. By creating an interactive, voice-powered learning companion, Crowbot helps kids build genuine communication skills, fosters curiosity through conversational education, and provides a healthy alternative to passive screen-based interactions.

![Project Idea](project.png)

### Vision & Mission

- **Reclaim Childhood**: Combat the graying of modern childhood by offering meaningful, interactive learning experiences.
- **AI for Good**: Demonstrate how AI can enhance human development rather than replace human connection.
- **Educational Innovation**: Bridge the gap between technology and education through natural voice interfaces.
- **IoT Accessibility**: Make advanced AI learning tools available through affordable, distributed IoT devices.

### Key Features & Benefits

- **Educational Motivation**: Transforms learning into playful conversations, making education enjoyable and less intimidating for children.
- **Voice-Powered Interaction**: Natural speech recognition and synthesis allow kids to communicate freely, building language skills and confidence.
- **IoT Integration**: Supports both local standalone operation and distributed MQTT-based processing for scalable IoT deployments.
- **Customizable Learning**: Flexible prompt system enables tailored educational content for different age groups and subjects.
- **Real-time Feedback**: Immediate AI responses help reinforce learning concepts through interactive dialogue.
- **Privacy-Focused**: Designed with children's data protection in mind, with local processing options available.

### Target Audience

- Children aged 5-12 been couries.
- Educators seeking innovative teaching tools
- Parents wanting to supplement traditional learning methods
- IoT enthusiasts building smart educational devices

## Architecture Diagram

```
[Local Device / IoT] <--> MQTT Broker <--> [Server] <--> HTTP API <--> [Web App]
     |                           |              |
     v                           v              v
[Audio Recording] --> [STT] --> [LLM] --> [TTS] --> [Playback]
     |                           |
     +--> Local Test (cmd/local) |
     +--> MQTT Test (test_mqtt.sh)
```

## Future Features

### Planned Iterations

- [x] MQTT Broker
- [x] Backend MQTT
- [x] Local Tests
- [x] Worfklow Simulation
- [x] Esp32 Setup with sensors (microphone, amplifier & speaker)
- [ ] Database Persistency
- [ ] HTTP Backend ApiRest
- [ ] Frontend Webapp Dashboard visualizer

## Architecture

### Components

- **Local Recorder** (`internal/utils/local_recorder.go`): Handles audio recording from microphone, saves to WAV, and triggers processing.
- **Local Orchestrator** (`internal/core/local_orchestrator.go`): Coordinates STT → LLM → TTS pipeline for local processing.
- **Orchestrator** (`internal/core/orchestrator.go`): General orchestrator for MQTT-based processing.
- **MQTT Client** (`internal/mqtt/client.go`): Handles MQTT connections, publishing, and subscribing for distributed processing.
- **Services**:
  - **GCP STT** (`internal/services/gcp_stt_service.go`): Converts audio to text using Google Speech-to-Text.
  - **Gemini LLM** (`internal/services/gemini_service.go`): Processes text prompts and generates responses.
  - **GCP TTS** (`internal/services/gcp_tts_service.go`): Converts text responses to audio and handles playback.

### Local Simulation Flow

1. **Recording**: User presses 'R' to start/stop recording via PortAudio.
2. **Audio Processing**: Raw PCM chunks are collected and saved as `local_record.wav`.
3. **STT**: WAV file is sent to Google STT (Spanish), returns transcribed text.
4. **LLM**: Text is sent to Gemini with system prompt, returns response text.
5. **TTS**: Response text is synthesized to audio (Spanish voice).
6. **Playback**: Audio is played via PortAudio, and saved as `local_response.wav` for debugging.

### MQTT Distributed Flow

1. **Audio Encoding**: Split recorded audio into chunks (e.g., 1-second PCM segments), encode to base64.
2. **MQTT Publish**: Send chunks to MQTT topics (`/device/audio/start`, `/device/audio/chunk`, `/device/audio/end`).
3. **Server Processing**: MQTT server subscribes, reassembles audio, runs STT/LLM/TTS, splits response into chunks.
4. **MQTT Subscribe**: Client receives response chunks from `/device/{device_id}/audio/response_chunk`, decodes, reassembles, and plays.
5. **Chunk Management**: Use sequence numbers and total count for ordering.

### Prerequisites

- Go 1.25.1+
- Google Cloud credentials
- Environment variables:
  - `GEMINI_API_KEY`: Your Gemini API key
  - `PROMPT`: System prompt for the LLM (e.g., "You are a helpful assistant.")
- Dependencies: Install with `go mod tidy`
- For MQTT: Mosquitto broker (see `infra/mosquitto/mosquitto.conf`)

### Running Local Simulation

```bash
go run cmd/local/main.go
```

- Press 'R' then Enter to start recording.
- Speak in Spanish.
- Press 'R' again to stop and process.
- Response audio plays automatically.

### Running MQTT Server

```bash
go run cmd/mqtt/main.go
```

- Starts MQTT server listening on `tcp://localhost:1883`.
- Processes audio chunks from devices and sends back responses.

### Testing MQTT

Use the provided test script:

```bash
./test_mqtt.sh test.wav [broker] [port]
```

- Converts `test.wav` to chunks, sends via MQTT, receives and plays response.

#### Files Generated

- `local_record.wav`: Recorded audio (local mode).
- `local_response.wav`: Synthesized response (local mode).
- `combined.pcm`: Reassembled response audio (MQTT mode).

This setup allows scaling to multiple IoT devices while keeping local fallback.</content>
<parameter name="filePath">README.md
