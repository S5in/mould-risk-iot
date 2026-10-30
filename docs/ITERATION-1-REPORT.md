# Iteration 1 report: Foundation

Student: Maksym · Project: Secure IoT Mould-Risk Monitoring Platform · Date: 30 Oct 2026 · Release: v0.1.0

## Goal
Deliver the foundation: repository, CI, Docker Compose with Mosquitto and TimescaleDB, an ESP32 that publishes with its own login, a Go service that stores readings, and a demo of live readings queried from the database.

## What worked
- Repository with CI (gofmt, vet, tests, firmware build, secret scanning).
- Docker Compose stack: TimescaleDB and Mosquitto with anonymous access off, a password per device, and a per-device topic ACL. Security tests showed wrong passwords and wrong-topic publishes are refused.
- Database schema with `org_id` on every org-owned table, a `readings` hypertable and CHECK constraints (a humidity of 150 is refused by the database itself).
- Go simulator publishing realistic readings with its own device login.
- Go ingestion service: subscribes, validates (topic shape, schema, ranges, timestamp, known device in that organisation), stores, and logs rejects without storing them. Unit tests cover validation and handler outcomes; a database test covers the SQL.
- Demo: live readings appear in the database and can be queried with SQL.

## What did not work / not done
- **The real ESP32 and SHT31 were not tested.** The Mouser order arrives 3 to 4 working days after the deadline. The firmware is written and compiled in CI, and the simulator stands in for it. The objective "ESP32 publishes with its own login" is therefore only partly met, and I am reporting that plainly.
- No TLS yet, and no fault-injection tests on hardware (Wi-Fi drop, broker restart).

## Problems and solutions
| Problem | Solution |
|---|---|
| Compose warnings and a truncated database password | A `$` in the password; switched to alphanumeric and recreated the empty volume |
| Mosquitto exited: "Unable to open pwfile" | Files were owned by root; chown to `mosquitto` and chmod 0600 |
| Tests found no package / nested folder | Fixed the layout and ran Go commands from the repo root |
| Time pressure at the deadline, hardware late | Simulator-first plan; honest reporting of the gap |

## What I learned
Authentication and ACLs are part of the design, not a later add-on; validating at the boundary and taking org and device identity from the database rather than the message prevents a whole class of tenant-mixing bugs.

## Next steps
- **Iteration 2 (due 4 Dec 2026):** flash the real node and re-run the data path on hardware; dew point and mould-risk indicator; Go REST API; React dashboard; login.
- **Iteration 3 (due 15 Feb 2027):** TLS, alerts, rate limits, audit of the security checklist.
- **Iteration 4 (due 16 Apr 2027):** evaluation.

