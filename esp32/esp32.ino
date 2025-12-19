#include <WiFi.h>
#include <WiFiManager.h>
#include <AsyncMQTT_ESP32.h>
#include <Ticker.h>
#include <driver/i2s.h>
#include "base64.hpp"

// ===== MQTT =====
const char* mqtt_server = "192.168.1.92";
const uint16_t mqtt_port = 1883;

const char* robot_id = "crowbot-el-primero";
const char* kid_id   = "KID001";

AsyncMqttClient mqttClient;
Ticker mqttReconnectTimer;

// ===== Pines I2S / DAC / Botón =====
#define I2S_WS   25
#define I2S_SD   33
#define I2S_SCK  32
#define I2S_PORT I2S_NUM_0
#define SWITCH_PIN 13
#define DAC_PIN 26

// ===== Audio =====
#define SAMPLE_RATE     16000
#define BITS_PER_SAMPLE 32

// Trabajamos por N MUESTRAS, no por bytes
#define CHUNK_SAMPLES 1024                         // ~64 ms de audio a 16 kHz
#define I2S_READ_BYTES (CHUNK_SAMPLES * 4)         // 32-bit mic -> 4 bytes por muestra
#define PCM_BYTES      (CHUNK_SAMPLES * 2)         // 16-bit PCM

// Buffers estáticos para evitar malloc en cada iteración
int32_t i2sRawBuffer[CHUNK_SAMPLES];              // 32-bit desde I2S
int16_t audioChunkBuffer[CHUNK_SAMPLES];          // 16-bit PCM
char    base64Buffer[3000];                       // suficiente para PCM_BYTES en base64

// Buffer dinámico solo para la respuesta de vuelta
int16_t* audioBuffer = nullptr;
size_t   bufferSize  = 0; // en muestras
int      totalChunks = 0;

bool recording = false;
bool playing   = false;

// ===== Prototipos MQTT =====
void connectToMqtt();
void onMqttConnect(bool sessionPresent);
void onMqttDisconnect(AsyncMqttClientDisconnectReason reason);
void onMqttMessage(char* topic, char* payload, AsyncMqttClientMessageProperties properties,
                   size_t len, size_t index, size_t total);

// ===== Prototipos app =====
void startRecording();
void stopRecording();
void sendAudioChunk();
void playAudio();

// SETUP 
void setup() {
  Serial.begin(115200);
  delay(1000);

  // --- wifimanager ---
  WiFiManager wifiManager;
  // degug: reset password
  // wifiManager.resetSettings();
  wifiManager.autoConnect("Configurar-Crowbot");

  Serial.println("WiFi connected!");
  Serial.print("IP: ");
  Serial.println(WiFi.localIP());

  // --- mqtt asincrono ---
  mqttClient.onConnect(onMqttConnect);
  mqttClient.onDisconnect(onMqttDisconnect);
  mqttClient.onMessage(onMqttMessage);
  mqttClient.setServer(mqtt_server, mqtt_port);

  connectToMqtt();

  // --- boton ---
  pinMode(SWITCH_PIN, INPUT_PULLUP);

  // --- I2S mic ---
    i2s_config_t i2s_config = {
    .mode = (i2s_mode_t)(I2S_MODE_MASTER | I2S_MODE_RX),
    .sample_rate = SAMPLE_RATE,
    .bits_per_sample = I2S_BITS_PER_SAMPLE_32BIT,
    .channel_format = I2S_CHANNEL_FMT_ONLY_LEFT,
    .communication_format = I2S_COMM_FORMAT_I2S_MSB,
    .intr_alloc_flags = 0,
    .dma_buf_count = 8,
    .dma_buf_len = 256,
    .use_apll = false,
    .tx_desc_auto_clear = false,
    .fixed_mclk = 0
};

  i2s_pin_config_t pin_config = {
    .bck_io_num   = I2S_SCK,
    .ws_io_num    = I2S_WS,
    .data_out_num = I2S_PIN_NO_CHANGE,
    .data_in_num  = I2S_SD
  };  

  i2s_driver_install(I2S_PORT, &i2s_config, 0, NULL);
  i2s_set_pin(I2S_PORT, &pin_config);
  i2s_set_clk(I2S_PORT, SAMPLE_RATE, I2S_BITS_PER_SAMPLE_32BIT, I2S_CHANNEL_MONO);
  Serial.println("ESP32 ready");
}

// LOOP
void loop() {
  // mqtt asincronico

  int switchState = digitalRead(SWITCH_PIN);

  if (switchState == LOW && !recording) {
    Serial.println("Switch pressed, starting recording");
    startRecording();
  } else if (switchState == HIGH && recording) {
    Serial.println("Switch released, stopping recording");
    stopRecording();
  }

  if (recording) {
    sendAudioChunk();
  }

  delay(10);
}

// MQTT

void connectToMqtt() {
  Serial.println("Connecting to MQTT...");
  mqttClient.connect();
}

void onMqttConnect(bool sessionPresent) {
  Serial.println("Connected to MQTT broker");

  // Suscripciones permanentes
  String responseChunkTopic = "/device/" + String(robot_id) + "/audio/response_chunk";
  String responseEndTopic   = "/device/" + String(robot_id) + "/audio/response_end";

  mqttClient.subscribe(responseChunkTopic.c_str(), 1);
  mqttClient.subscribe(responseEndTopic.c_str(), 1);

  Serial.printf("Subscribed to %s and %s\n",
                responseChunkTopic.c_str(), responseEndTopic.c_str());
}

void onMqttDisconnect(AsyncMqttClientDisconnectReason reason) {
  Serial.printf("Disconnected from MQTT, reason=%d\n", (int)reason);
  mqttReconnectTimer.once(2, connectToMqtt);
}

void onMqttMessage(char* topic, char* payload,
                   AsyncMqttClientMessageProperties properties,
                   size_t len, size_t index, size_t total) {

  String msg;
  msg.reserve(len);
  for (size_t i = 0; i < len; i++) {
    msg += (char)payload[i];
  }

  Serial.printf("MQTT received on %s: %s\n", topic, msg.c_str());

  String t = String(topic);
  if (t.endsWith("/response_end")) {
    Serial.println("Response end received");
    playAudio();
  } else if (t.endsWith("/response_chunk")) {
    int indexStart = msg.indexOf("\"index\":") + 8;
    int indexEnd   = msg.indexOf(",", indexStart);
    int chunkIdx   = msg.substring(indexStart, indexEnd).toInt();

    int totalStart = msg.indexOf("\"total\":") + 8;
    int totalEnd   = msg.indexOf(",", totalStart);
    totalChunks    = msg.substring(totalStart, totalEnd).toInt();

    int dataStart  = msg.indexOf("\"data\":\"") + 8;
    int dataEnd    = msg.indexOf("\"", dataStart);
    String data    = msg.substring(dataStart, dataEnd);

    // decodificar base64
    size_t maxDecodedLen = (data.length() * 3) / 4 + 1;
    uint8_t* decoded = (uint8_t*)malloc(maxDecodedLen);
    if (!decoded) {
      Serial.println("Failed to alloc decoded buffer");
      return;
    }

    size_t decodedLen = decode_base64((unsigned char*)data.c_str(), decoded);
    Serial.printf("Decoded %d bytes from base64\n", decodedLen);

    size_t samples = decodedLen / 2;
    audioBuffer = (int16_t*)realloc(audioBuffer,
                                    (bufferSize + samples) * sizeof(int16_t));
    memcpy(audioBuffer + bufferSize, decoded, decodedLen);
    bufferSize += samples;

    free(decoded);
    Serial.printf("Chunk %d/%d received, total buffer size: %d samples\n",
                  chunkIdx + 1, totalChunks, bufferSize);
  }
}

// LOGICA DE AUDIO 

void startRecording() {
  recording = true;

  if (!mqttClient.connected()) {
    connectToMqtt();
  }

  Serial.println("Recording started");
  i2s_start(I2S_PORT);
  Serial.println("I2S started");

  // mensaje de inicio
  String startMsg = "{\"robot_id\": \"" + String(robot_id) +
                    "\", \"kid_id\": \"" + String(kid_id) + "\"}";
  mqttClient.publish("/device/audio/start", 1, false,
                     startMsg.c_str(), startMsg.length());
  Serial.println("Published start message");

  // reset buffer de respuesta
  if (audioBuffer) {
    free(audioBuffer);
    audioBuffer = nullptr;
  }
  bufferSize = 0;
}

void stopRecording() {
  recording = false;
  Serial.println("Recording stopped");

  mqttClient.publish("/device/audio/end", 1, false, "done", 4);
  Serial.println("Published end message");

  i2s_stop(I2S_PORT);
  Serial.println("I2S stopped");
}

void sendAudioChunk() {
  if (!mqttClient.connected()) {
    Serial.println("MQTT not connected, skipping chunk send");
    return;
  }

  size_t bytesRead = 0;
  esp_err_t result = i2s_read(I2S_PORT,
                              (void*)i2sRawBuffer,
                              I2S_READ_BYTES,
                              &bytesRead,
                              pdMS_TO_TICKS(50));

  if (result == ESP_OK && bytesRead > 0) {
    size_t samples = bytesRead / 4; // 32-bit -> 4 bytes

      // convertir 32-bit I2S (INMP441 entrega 24 bits útiles) a 16-bit PCM limpio
      for (size_t i = 0; i < samples; i++) {
          // quitar los 8 bits menos significativos (I2S es 24 bits válidos dentro de 32)
          int32_t s32 = i2sRawBuffer[i] >> 8;

          // aplicar ganancia (x4 o x8)
          s32 *= 8;  // puedes probar 4, 8, 16

          // Paso 3: clip para evitar distorsión
          if (s32 > 32767)  s32 = 32767;
          if (s32 < -32768) s32 = -32768;

          // guardar en tu buffer 16-bit  
          audioChunkBuffer[i] = (int16_t)s32;
      }

    size_t pcmBytes = samples * 2;

    // debug: nivel de audio
    long sum = 0;
    for (size_t i = 0; i < samples; i++) {
      sum += abs(audioChunkBuffer[i]);
    }
    long avg = sum / samples;
    Serial.printf("Read %d bytes from I2S, samples=%d, Audio level=%ld\n",
                  bytesRead, samples, avg);

    // base64
    unsigned int encoded_len = encode_base64(
      (unsigned char*)audioChunkBuffer,
      pcmBytes,
      (unsigned char*)base64Buffer
    );

    Serial.printf("Encoded to %u base64 chars\n", encoded_len);

    // publicar chunk (payload = const char*)
    uint16_t packetId = mqttClient.publish(
      "/device/audio/chunk",
      1,      // QoS 1
      false,  // retain
      base64Buffer,
      encoded_len
    );

    if (packetId == 0) {
      Serial.println("Failed to publish chunk (packetId=0)");
    } else {
      Serial.printf("Chunk published, packetId=%u\n", packetId);
    }

  } else {
    Serial.printf("I2S read failed or no data: result=%d, bytesRead=%d\n",
                  (int)result, (int)bytesRead);
  }
}

void playAudio() {
  if (!audioBuffer || bufferSize == 0) {
    Serial.println("No audio to play");
    return;
  }

  playing = true;
  Serial.printf("Starting playback of %d samples\n", bufferSize);

  for (size_t i = 0; i < bufferSize; i++) {
    int16_t sample = audioBuffer[i];
    uint8_t val = (sample + 32768) >> 8; // 16b signed -> 8b unsigned
    dacWrite(DAC_PIN, val);
    delayMicroseconds(62); // ~16 kHz
  }

  free(audioBuffer);
  audioBuffer = nullptr;
  bufferSize = 0;
  playing = false;
  Serial.println("Playback finished");
}
