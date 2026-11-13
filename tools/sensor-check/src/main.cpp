#include <Adafruit_SHT31.h>
#include <Wire.h>

Adafruit_SHT31 sht31;
bool sensorOk = false;

void setup() {
  Serial.begin(115200);
  delay(1000);
  Wire.begin();  // SDA = GPIO21, SCL = GPIO22
  Serial.println("Scanning I2C bus...");
  int found = 0;
  for (uint8_t addr = 1; addr < 127; addr++) {
    Wire.beginTransmission(addr);
    if (Wire.endTransmission() == 0) {
      Serial.printf("  device found at 0x%02X\n", addr);
      found++;
    }
  }
  if (found == 0) Serial.println("  nothing found: check wires");
  if (sht31.begin(0x44)) { Serial.println("SHT31 started at 0x44"); sensorOk = true; }
  else if (sht31.begin(0x45)) { Serial.println("SHT31 started at 0x45"); sensorOk = true; }
  else Serial.println("SHT31 not found at 0x44 or 0x45");
}

void loop() {
  if (sensorOk) {
    float t = sht31.readTemperature();
    float h = sht31.readHumidity();
    if (isnan(t) || isnan(h)) Serial.println("read failed");
    else Serial.printf("temperature %.2f C   humidity %.2f %%RH\n", t, h);
  }
  delay(2000);
}
