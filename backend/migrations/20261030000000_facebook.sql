-- Fase 1 de redes: una página de Facebook por sitio. Cada artículo que pasa a publicado (autopiloto o a mano) se comparte una vez.
CREATE TABLE IF NOT EXISTS site_facebook (
 site_id uuid PRIMARY KEY REFERENCES sites(id) ON DELETE CASCADE,
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 status text NOT NULL DEFAULT 'choosing' CHECK (status IN ('choosing','connected','broken')),
 page_id text NOT NULL DEFAULT '',
 page_name text NOT NULL DEFAULT '',
 token_enc bytea,               -- token de la página, cifrado
 choices_enc bytea,             -- páginas por elegir justo después de autorizar (cifradas, con sus tokens)
 choices_until timestamptz,
 last_error text,
 connected_at timestamptz
);
CREATE TABLE IF NOT EXISTS social_posts (
 article_id uuid PRIMARY KEY REFERENCES articles(id) ON DELETE CASCADE,
 site_id uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 page_id text NOT NULL,
 status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sending','sent','failed','unknown','cancelled')),
 attempts int NOT NULL DEFAULT 0,
 post_id text,
 last_error text,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS social_posts_pending ON social_posts(updated_at) WHERE status='pending';
-- El disparador cubre todas las rutas de publicación sin repetir lógica en Go. Solo entra lo publicado después de conectar.
CREATE OR REPLACE FUNCTION faro_queue_social() RETURNS trigger AS $$
BEGIN
 IF NEW.status = 'published' AND (TG_OP = 'INSERT' OR OLD.status IS DISTINCT FROM 'published') THEN
  INSERT INTO social_posts(article_id, site_id, tenant_id, page_id)
  SELECT NEW.id, NEW.site_id, NEW.tenant_id, f.page_id FROM site_facebook f WHERE f.site_id = NEW.site_id AND f.status = 'connected'
  ON CONFLICT DO NOTHING;
 END IF;
 RETURN NEW;
END $$ LANGUAGE plpgsql;
-- Las migraciones corren en cada arranque: crear el disparador solo si falta (recrearlo bloquea articles y choca con el autopiloto).
DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'articles_social') THEN
  CREATE TRIGGER articles_social AFTER INSERT OR UPDATE OF status ON articles FOR EACH ROW EXECUTE FUNCTION faro_queue_social();
 END IF;
END $$;
