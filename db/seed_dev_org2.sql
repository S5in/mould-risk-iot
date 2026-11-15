-- Dev data: a second organisation with its own property, room, simulated
-- device and a few readings. Used to prove that a user of one organisation
-- cannot see another organisation's data. Fake names only. Safe to run twice.
INSERT INTO organisations (slug, name)
VALUES ('other-org', 'Other Housing Body')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO properties (org_id, name)
SELECT o.id, 'Other House'
FROM organisations o
WHERE o.slug = 'other-org'
  AND NOT EXISTS (SELECT 1 FROM properties p WHERE p.org_id = o.id AND p.name = 'Other House');

INSERT INTO rooms (org_id, property_id, name)
SELECT p.org_id, p.id, 'Kitchen'
FROM properties p
JOIN organisations o ON o.id = p.org_id
WHERE o.slug = 'other-org' AND p.name = 'Other House'
  AND NOT EXISTS (SELECT 1 FROM rooms r WHERE r.property_id = p.id AND r.name = 'Kitchen');

INSERT INTO devices (org_id, room_id, device_key, is_simulated)
SELECT r.org_id, r.id, 'sim-002', true
FROM rooms r
JOIN properties p ON p.id = r.property_id
JOIN organisations o ON o.id = r.org_id
WHERE o.slug = 'other-org' AND p.name = 'Other House' AND r.name = 'Kitchen'
ON CONFLICT (org_id, device_key) DO NOTHING;

-- Twelve readings, one every 30 minutes up to now, so history charts have something to show.
INSERT INTO readings (device_id, org_id, room_id, device_time, temperature_c, humidity_rh)
SELECT d.id, d.org_id, d.room_id,
       date_trunc('second', now()) - g * interval '30 minutes',
       19 + (g % 3),
       62 + (g % 5) * 3
FROM devices d
JOIN organisations o ON o.id = d.org_id
CROSS JOIN generate_series(0, 11) AS g
WHERE o.slug = 'other-org' AND d.device_key = 'sim-002'
ON CONFLICT DO NOTHING;