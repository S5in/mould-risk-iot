# Secure IoT Mould-Risk Monitoring Platform

ESP32 + SHT31 sensor node -> Wi-Fi -> MQTT (Mosquitto) -> Go ingestion -> PostgreSQL/TimescaleDB. Later iterations add a REST API and React dashboard. The mould-risk output is an indicator, not a diagnosis.

Status: **v0.1.0, Iteration 1 (foundation)**. Verified with a simulator; real hardware pending.

## Architecture
```
sensor node / simulator --MQTT (own login, ACL)--> Mosquitto --> ingestion (validate) --> TimescaleDB
        org/{org}/device/{device}/telemetry
```

## Message contract
Topic: `org/{org-slug}/device/{device-key}/telemetry`. Payload (max 1024 bytes, no extra fields):

    {"temperature_c": 21.5, "humidity_rh": 55.2, "ts": "2026-10-30T11:59:50Z"}

Rules: temperature -40 to 85 C, humidity 0 to 100 %, `ts` not more than 5 minutes in the future. The org and device come from the topic and must exist in the database; `org_id`, room and device ids are taken from the database, never from the message.

## Run it
Needs Docker Desktop and Go.
1. Copy `.env.example` to `.env`; set alphanumeric passwords (never commit `.env`).
2. Create broker users (hashes go in `mosquitto/passwd`, git-ignored) for `sim-001` and `ingest`, then fix ownership as in your setup notes.
3. `docker compose up -d --build`
4. Load schema and seed: `db/migrations/*.sql`, `db/seed_dev.sql`.
5. `$env:SIM_MQTT_PASSWORD="<sim-001 password>"; go run ./services/simulator -interval 2s`
6. Query: `docker compose exec db psql -U <user> -d <db> -c "select * from readings order by device_time desc limit 5;"`

## Tests
`go test ./...` (see docs/TESTING.md). CI runs on every push.

## Firmware
`firmware/` is a PlatformIO project. Copy `include/secrets.h.example` to `include/secrets.h` (git-ignored), fill in values, then `pio run -t upload`. Wiring: SHT31 SDA to GPIO21, SCL to GPIO22, 3V3, GND.

## Security notes
No anonymous MQTT; one login per device; per-device topic ACL; input validated at the boundary; secrets only in git-ignored files; secret scanning in CI. No TLS yet (Iteration 3).

## Licence
See LICENSE.
