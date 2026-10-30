// Mould-risk sensor node: reads an SHT31 and publishes JSON to MQTT.
#include <Adafruit_SHT31.h>
#include <ArduinoJson.h>
#include <PubSubClient.h>
#include <WiFi.h>
#include <Wire.h>
#include <time.h>

#include "secrets.h"

static const unsigned long PUBLISH_INTERVAL_MS = 30000;
static const unsigned long RETRY_INTERVAL_MS = 5000;

Adafruit_SHT31 sht31;
WiFiClient wifiClient;
PubSubClient mqtt(wifiClient);

char topic[96];
unsigned long lastPublish = 0;
unsigned long lastRetry = 0;

static void connectWifi() {
  if (WiFi.status() == WL_CONNECTED) return;
  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
  Serial.print("Wi-Fi connecting");
  const unsigned long start = millis();
  while (WiFi.status() != WL_CONNECTED && millis() - start < 15000) {
    delay(500);
    Serial.print(".");
  }
  Serial.println(WiFi.status() == WL_CONNECTED ? " connected" : " failed");
}

static bool timeIsSet() { return time(nullptr) > 1700000000; }

static void connectMqtt() {
  if (mqtt.connected() || WiFi.status() != WL_CONNECTED) return;
  Serial.print("MQTT connecting... ");
  if (mqtt.connect(DEVICE_KEY, DEVICE_KEY, MQTT_PASSWORD)) {
    Serial.println("connected");
  } else {
    Serial.printf("failed, rc=%d\n", mqtt.state());
  }
}

static bool isoTimestamp(char *out, size_t n) {
  time_t now = time(nullptr);
  struct tm tm;
  gmtime_r(&now, &tm);
  return strftime(out, n, "%Y-%m-%dT%H:%M:%SZ", &tm) > 0;
}

static void publishReading() {
  const float t = sht31.readTemperature();
  const float h = sht31.readHumidity();
  if (isnan(t) || isnan(h)) {
    Serial.println("sensor read failed, skipping");
    return;
  }
  if (!timeIsSet()) {
    Serial.println("clock not set yet, skipping");
    return;
  }
  char ts[25];
  isoTimestamp(ts, sizeof ts);

  JsonDocument doc;
  doc["temperature_c"] = serialized(String(t, 1));
  doc["humidity_rh"] = serialized(String(h, 1));
  doc["ts"] = ts;
  char payload[160];
  serializeJson(doc, payload, sizeof payload);

  if (mqtt.publish(topic, payload)) {
    Serial.printf("published %s\n", payload);
  } else {
    Serial.println("publish failed");
  }
}

void setup() {
  Serial.begin(115200);
  snprintf(topic, sizeof topic, "org/%s/device/%s/telemetry", ORG_SLUG, DEVICE_KEY);

  Wire.begin();  // SDA GPIO21, SCL GPIO22
  if (!sht31.begin(0x44)) {
    Serial.println("SHT31 not found at 0x44 (try 0x45)");
  }
  connectWifi();
  configTime(0, 0, "pool.ntp.org", "time.google.com");
  mqtt.setServer(MQTT_HOST, MQTT_PORT);
}

void loop() {
  const unsigned long now = millis();
  if (WiFi.status() != WL_CONNECTED || !mqtt.connected()) {
    if (now - lastRetry >= RETRY_INTERVAL_MS) {
      lastRetry = now;
      connectWifi();
      connectMqtt();
    }
  } else {
    mqtt.loop();
    if (now - lastPublish >= PUBLISH_INTERVAL_MS) {
      lastPublish = now;
      publishReading();
    }
  }
}
