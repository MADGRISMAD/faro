-- Direcciones anteriores de un artículo (por ejemplo cuando se quitaron los acentos de los slugs): se siguen resolviendo.
CREATE TABLE IF NOT EXISTS article_slug_aliases (
 site_id uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
 old_slug text NOT NULL,
 article_id uuid NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
 PRIMARY KEY (site_id, old_slug)
);
