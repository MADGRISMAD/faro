ALTER TABLE sites ADD COLUMN IF NOT EXISTS research jsonb;
ALTER TABLE sites ADD COLUMN IF NOT EXISTS autopilot boolean NOT NULL DEFAULT false;
ALTER TABLE sites ADD COLUMN IF NOT EXISTS posts_per_week integer NOT NULL DEFAULT 3 CHECK(posts_per_week BETWEEN 1 AND 14);
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS kind text NOT NULL DEFAULT 'article' CHECK(kind IN ('article','research'));
ALTER TABLE sites ADD COLUMN IF NOT EXISTS auto_publish boolean NOT NULL DEFAULT true;
ALTER TABLE sites ALTER COLUMN posts_per_week SET DEFAULT 2;
