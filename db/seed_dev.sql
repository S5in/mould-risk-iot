INSERT INTO organisations (slug, name) VALUES ('demo-org', 'Demo Organisation');

INSERT INTO properties (org_id, name)
SELECT id, 'Demo House' FROM organisations WHERE slug = 'demo-org';

INSERT INTO rooms (org_id, property_id, name)
SELECT p.org_id, p.id, 'Living room' FROM properties p WHERE p.name = 'Demo House';

INSERT INTO devices (org_id, room_id, device_key, is_simulated)
SELECT r.org_id, r.id, 'sim-001', true FROM rooms r WHERE r.name = 'Living room';