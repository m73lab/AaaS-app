-- AaaS Dashboard Schema (Supabase Cloud)
-- Execute via SQL Editor in Supabase Dashboard

-- Tenants
CREATE TABLE IF NOT EXISTS tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL,
  plan TEXT NOT NULL DEFAULT 'free',
  rate_limit_per_min INT NOT NULL DEFAULT 100,
  rate_limit_per_hour INT NOT NULL DEFAULT 1000,
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- API Keys (hashed with SHA-256)
CREATE TABLE IF NOT EXISTS api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  key_hash TEXT NOT NULL UNIQUE,
  key_prefix TEXT NOT NULL,
  name TEXT,
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_used_at TIMESTAMPTZ
);

-- Usage logs (one row per proxy request)
CREATE TABLE IF NOT EXISTS usage_logs (
  id BIGSERIAL PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
  session_id TEXT NOT NULL,
  model TEXT,
  provider TEXT,
  format TEXT,
  prompt_tokens INT,
  completion_tokens INT,
  entities_detected INT,
  categories JSONB,
  action TEXT NOT NULL,
  latency_ms INT
);

-- Rate limit events (audit trail)
CREATE TABLE IF NOT EXISTS rate_limit_events (
  id BIGSERIAL PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
  window_type TEXT NOT NULL,
  limit_val INT NOT NULL,
  current_val INT NOT NULL
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_usage_logs_tenant ON usage_logs(tenant_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_usage_logs_timestamp ON usage_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rate_limit_events_tenant ON rate_limit_events(tenant_id, timestamp DESC);

-- Updated_at trigger
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tenants_updated_at
  BEFORE UPDATE ON tenants
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Seed demo tenant
INSERT INTO tenants (name, email, plan, rate_limit_per_min, rate_limit_per_hour)
VALUES ('Demo', 'demo@aas.com', 'free', 100, 1000)
ON CONFLICT (name) DO NOTHING;

-- Admins (dashboard operators)
CREATE TABLE IF NOT EXISTS admins (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  name TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed default admin (password: admin123 — change immediately)
-- hash generated with: crypto.createHash('sha256').update('admin123').digest('hex')
INSERT INTO admins (email, password_hash, name)
VALUES ('admin@aas.com', '240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9', 'Default Admin')
ON CONFLICT (email) DO NOTHING;
