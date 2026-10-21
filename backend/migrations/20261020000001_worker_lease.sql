ALTER TABLE jobs ADD COLUMN IF NOT EXISTS reserved_microusd bigint NOT NULL DEFAULT 0;
ALTER TABLE usage_events ADD COLUMN IF NOT EXISTS status text NOT NULL DEFAULT 'measured';
CREATE UNIQUE INDEX IF NOT EXISTS sites_domain_unique ON sites(domain);
