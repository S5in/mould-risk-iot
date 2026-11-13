BEGIN;
INSERT INTO rooms (org_id, property_id, name)
SELECT r.org_id, r.property_id, 'Bedroom'
FROM rooms r
WHERE r.name = 'Living room'
  AND NOT EXISTS (SELECT 1 FROM rooms WHERE name = 'Bedroom');
INSERT INTO devices (org_id, room_id, device_key, is_simulated)
SELECT r.org_id, r.id, 'esp32-001', false
FROM rooms r
WHERE r.name = 'Bedroom'
ON CONFLICT (org_id, device_key) DO NOTHING;
COMMIT;
SELECT o.slug, r.name AS room, d.device_key, d.is_simulated
FROM devices d
JOIN organisations o ON o.id = d.org_id
LEFT JOIN rooms r ON r.id = d.room_id
ORDER BY d.device_key;
