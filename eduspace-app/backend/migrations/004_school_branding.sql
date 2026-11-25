-- Centralised per-school identity. One record lives on the tenant itself so
-- every role dashboard and generated document can resolve the same branding.
ALTER TABLE institutes
    ADD COLUMN IF NOT EXISTS logo_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS motto VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS address TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS phone VARCHAR(40) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS contact_email VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS website VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS primary_color VARCHAR(7) NOT NULL DEFAULT '#173653',
    ADD COLUMN IF NOT EXISTS secondary_color VARCHAR(7) NOT NULL DEFAULT '#168c8c';
