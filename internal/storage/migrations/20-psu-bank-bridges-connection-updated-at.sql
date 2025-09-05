ALTER TABLE psu_bank_bridge_connections  ADD COLUMN IF NOT EXISTS updated_at timestamp without time zone;
UPDATE psu_bank_bridge_connections SET updated_at = created_at WHERE updated_at IS NULL;
ALTER TABLE psu_bank_bridge_connections ALTER COLUMN updated_at SET NOT NULL;