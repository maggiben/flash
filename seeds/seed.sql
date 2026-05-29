-- Seed data for review (fixed UUIDs for reproducible demos)

INSERT INTO items (id, name, total_quantity, reserved_quantity) VALUES
    ('11111111-1111-1111-1111-111111111101', 'Limited Edition Sneakers', 25, 0),
    ('11111111-1111-1111-1111-111111111102', 'Vintage Vinyl Box Set', 10, 0),
    ('11111111-1111-1111-1111-111111111103', 'Smart Watch Pro', 50, 0),
    ('11111111-1111-1111-1111-111111111104', 'Flash Sale Hoodie', 100, 0),
    ('11111111-1111-1111-1111-111111111105', 'Collector Pin (Last One)', 1, 0)
ON CONFLICT (id) DO NOTHING;
