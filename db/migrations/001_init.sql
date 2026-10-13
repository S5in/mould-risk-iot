CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE organisations (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       text NOT NULL UNIQUE,
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE properties (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     uuid NOT NULL REFERENCES organisations(id),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE rooms (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      uuid NOT NULL REFERENCES organisations(id),
    property_id uuid NOT NULL REFERENCES properties(id),
    name        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE devices (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       uuid NOT NULL REFERENCES organisations(id),
    room_id      uuid REFERENCES rooms(id),
    device_key   text NOT NULL,
    is_simulated boolean NOT NULL DEFAULT false,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (org_id, device_key)
);

CREATE TABLE readings (
    device_id     uuid NOT NULL REFERENCES devices(id),
    org_id        uuid NOT NULL REFERENCES organisations(id),
    room_id       uuid REFERENCES rooms(id),
    device_time   timestamptz NOT NULL,
    received_at   timestamptz NOT NULL DEFAULT now(),
    temperature_c double precision NOT NULL CHECK (temperature_c BETWEEN -40 AND 85),
    humidity_rh   double precision NOT NULL CHECK (humidity_rh BETWEEN 0 AND 100),
    PRIMARY KEY (device_id, device_time)
);

SELECT create_hypertable('readings', 'device_time');