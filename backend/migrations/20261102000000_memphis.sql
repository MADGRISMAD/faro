-- Estilo Memphis corporativo como predeterminado. Los sitios que tenían el estilo anterior por omisión pasan a él
-- UNA sola vez (las migraciones corren en cada arranque): quien elija «Limpio» después lo conserva.
ALTER TABLE sites ALTER COLUMN image_style SET DEFAULT 'memphis';
CREATE TABLE IF NOT EXISTS faro_once (name text PRIMARY KEY, at timestamptz NOT NULL DEFAULT now());
WITH first AS (INSERT INTO faro_once(name) VALUES ('memphis-default') ON CONFLICT DO NOTHING RETURNING 1)
UPDATE sites SET image_style='memphis' WHERE image_style='clean' AND EXISTS (SELECT 1 FROM first);
