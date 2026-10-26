CREATE TABLE IF NOT EXISTS article_images (
 article_id uuid PRIMARY KEY REFERENCES articles(id) ON DELETE CASCADE,
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 mime text NOT NULL,
 data bytea NOT NULL,
 alt text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE articles ADD COLUMN IF NOT EXISTS has_image boolean NOT NULL DEFAULT false;
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_kind_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_kind_check CHECK(kind IN ('article','research','image'));
ALTER TABLE sites ADD COLUMN IF NOT EXISTS images_enabled boolean NOT NULL DEFAULT true;
ALTER TABLE sites ADD COLUMN IF NOT EXISTS image_style text NOT NULL DEFAULT 'clean';
ALTER TABLE sites ADD COLUMN IF NOT EXISTS image_notes text NOT NULL DEFAULT '';
ALTER TABLE sites ADD COLUMN IF NOT EXISTS avoid_topics text NOT NULL DEFAULT '';
