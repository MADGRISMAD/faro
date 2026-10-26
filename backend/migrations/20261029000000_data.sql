-- Caché de métricas de palabras clave (compartida entre clientes: el volumen de una palabra no depende de quién pregunta).
CREATE TABLE IF NOT EXISTS keyword_cache (
 keyword text NOT NULL,
 location int NOT NULL,
 language text NOT NULL,
 volume int,
 competition text,
 competition_index int,
 cpc numeric,
 difficulty int,
 fetched_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (keyword, location, language)
);
-- Conexión de un sitio con Google Search Console. El refresh token se guarda cifrado (AES-GCM).
CREATE TABLE IF NOT EXISTS site_google (
 site_id uuid PRIMARY KEY REFERENCES sites(id) ON DELETE CASCADE,
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 refresh_enc bytea NOT NULL,
 property text NOT NULL DEFAULT '',
 connected_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS analytics_cache (
 site_id uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
 range text NOT NULL,
 body jsonb NOT NULL,
 fetched_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (site_id, range)
);
