# Test documentation (Iteration 1)

## Automated tests
Run from the repo root: `go test ./...`

| Package | What it checks | Needs services? |
|---|---|---|
| internal/telemetry | Topic parsing and reading validation: missing fields, unknown fields, temperature and humidity ranges, future timestamps, oversized payloads, bad JSON (12 cases) | No |
| internal/ingest | Handler outcomes with a fake store: stored, rejected (bad topic, bad JSON, out of range, missing field, future time, unknown device), duplicate, database failure; topic identity is passed to the store | No |
| internal/store | Real SQL against the database: insert, duplicate, unknown device, device under the wrong org | Yes: set `TEST_DB=1` and PGHOST, PGUSER, PGPASSWORD, PGDATABASE |

To run the database test against the Compose database:

    $env:TEST_DB="1"; $env:PGHOST="127.0.0.1"; $env:PGUSER="mould"; $env:PGDATABASE="mould"
$env:PGPASSWORD=((Get-Content .env | Where-Object { $_ -like 'POSTGRES_PASSWORD=*' }) -replace '^POSTGRES_PASSWORD=','')
go test ./internal/store -v

Compose does not publish a Postgres port unless your docker-compose.yml maps 5432; if it does not, the CI run (GitHub Actions, which uses a Timescale service container) is the evidence.

CI (`.github/workflows/ci.yml`) runs gofmt, go vet, migrations plus seed, go test with the database test enabled, a PlatformIO firmware build, and gitleaks secret scanning.

## Manual security tests (evidence for objective O6, first part)
| # | Test | Expected | Result |
|---|---|---|---|
| S1 | Connect with no username | Refused | evidence/testing1.png |
| S2 | Connect as sim-001 with a wrong password | `not Authorized` | evidence/testing1.png |
| S3 | sim-001 publishes to another device's topic | Message dropped by ACL | evidence/testing1.png |
| S4 | ingest tries to publish | Dropped by ACL | evidence/testing1.png |
| S5 | Valid reading via simulator | Row in `readings` | evidence/testing1.pngd |
| S6 | `simulator -bad` (humidity 150) | Ingestion logs `rejected`, no row | evidence/testing1.png |
| S7 | Unknown device key under a valid org | Rejected, no row | evidence/testing1.png |

## Data-path demo test
1. `docker compose up -d --build`
2. `go run ./services/simulator -count 5 -interval 2s`
3. `docker compose logs ingestion` shows `stored` lines.
4. `docker compose exec db psql -U <user> -d <db> -c "select device_time, temperature_c, humidity_rh from readings order by device_time desc limit 5;"`

## Known gaps (honest list)
- Wi-Fi loss recovery (H5) has not been tested on hardware yet.
- No local buffer: readings taken while Wi-Fi or the broker is down are lost.
- No TLS yet (planned Iteration 3). Passwords travel unencrypted on the local network.
- Broker restart and database outage behaviour are covered by reconnect code but not by an automated test.

## Hardware tests (Iteration 2, real node esp32-001)
Setup: ESP32-DevKitC-32E and DFRobot SHT31 on the home Wi-Fi, publishing every 30 s to Mosquitto on the development laptop, stored by the ingestion service. Plain MQTT on the home network; TLS is planned for Iteration 3.

| # | Test | What I did | Result | Evidence |
|---|---|---|---|---|
| H1 | Sensor check | Flashed tools/sensor-check (no network) | Temperature and humidity printed every 2 s (for example 16.05 C, 61.88 %RH) | evidence/testing2.png |
| H2 | End to end | Flashed the firmware, queried the database | Rows for esp32-001 stored; device_time and received_at about 50 ms apart | evidence/testing2.png |
| H3 | Live response | Breathed on and handled the sensor | Humidity rose to 68.9 %RH and temperature to 20.3 C in stored rows, then fell back | evidence/testing2.png |
| H4 | Broker restart | docker compose restart mosquitto while running | ESP32 logged rc=-4, retried about 5 s later and reconnected without a reset; readings resumed | evidence/testing4.png |
| H5 | Wi-Fi loss | Not tested yet | n/a | n/a |
| H6 | Wrong password | Changed the device password on the broker, restarted it | ESP32 logged rc=5 repeatedly, broker logged "not authorised", nothing stored; real password restored afterwards | evidence/testing6.png |
| H7 | Sensor unplugged | Pulled the SDA wire while running | "sensor read failed, skipping" twice, no rows stored for that period; recovered on the next tick after replugging | evidence/testing7.png |
