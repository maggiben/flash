-- Allow confirmed status (two-phase reserve → confirm lifecycle)

ALTER TABLE reservations DROP CONSTRAINT IF EXISTS reservations_status_check;
ALTER TABLE reservations ADD CONSTRAINT reservations_status_check
    CHECK (status IN ('active', 'confirmed', 'released', 'expired'));
