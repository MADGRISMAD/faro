-- Los sitios que ya existen (y los dados de alta a mano) quedan confirmados; los que crea la IA por URL esperan a que el cliente revise su Brand DNA.
ALTER TABLE sites ADD COLUMN IF NOT EXISTS brand_confirmed_at timestamptz DEFAULT now();
