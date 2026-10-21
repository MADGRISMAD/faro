CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE tenants (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 name text NOT NULL,
 budget_microusd bigint NOT NULL DEFAULT 500000 CHECK (budget_microusd >= 0),
 spent_microusd bigint NOT NULL DEFAULT 0 CHECK (spent_microusd >= 0),
 reserved_microusd bigint NOT NULL DEFAULT 0 CHECK (reserved_microusd >= 0),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE users (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 email text NOT NULL UNIQUE,
 password_hash text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE sessions (
 id_hash text PRIMARY KEY,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 csrf_hash text NOT NULL,
 expires_at timestamptz NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE sites (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 name text NOT NULL,
 domain text NOT NULL,
 canonical_url text NOT NULL,
 blog_prefix text NOT NULL DEFAULT '/blog',
 brand_description text NOT NULL DEFAULT '',
 tone text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(tenant_id, domain)
);
CREATE TABLE articles (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 site_id uuid NOT NULL REFERENCES sites(id),
 job_id uuid UNIQUE,
 title text NOT NULL,
 slug text NOT NULL,
 meta_description text NOT NULL,
 summary text NOT NULL,
 content jsonb NOT NULL,
 status text NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','approved','published')),
 published_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(site_id, slug)
);
CREATE TABLE jobs (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 site_id uuid NOT NULL REFERENCES sites(id),
 idempotency_key text NOT NULL,
 topic text NOT NULL,
 business text NOT NULL,
 audience text NOT NULL,
 keyword text,
 status text NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','running','done','failed')),
 attempts integer NOT NULL DEFAULT 0 CHECK(attempts BETWEEN 0 AND 2),
 article_id uuid REFERENCES articles(id),
 error_message text,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(tenant_id,idempotency_key)
);
ALTER TABLE articles ADD CONSTRAINT articles_job_id_fkey FOREIGN KEY(job_id) REFERENCES jobs(id);
CREATE UNIQUE INDEX one_running_job_per_tenant ON jobs(tenant_id) WHERE status='running';
CREATE TABLE usage_events (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 job_id uuid NOT NULL REFERENCES jobs(id),
 attempt integer NOT NULL,
 model text NOT NULL,
 input_tokens integer NOT NULL,
 output_tokens integer NOT NULL,
 cost_microusd bigint NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(job_id,attempt)
);
CREATE INDEX jobs_pending ON jobs(created_at) WHERE status='queued';
CREATE INDEX articles_published ON articles(site_id,published_at DESC) WHERE status='published';
