import { Layout } from '../components/Layout';

const PROXY = `curl https://your-edge.example/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "X-LLM-API-Key: $OPENAI_KEY" \\
  -H "X-LLM-Base-URL: https://api.openai.com/v1" \\
  -H "X-LLM-Format: openai" \\
  -H "X-Tenant-ID: demo" \\
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role":"user","content":"Email maria@acme.com about account 4111 1111 1111 1111"}]
  }'`;

const DASH = `curl https://dash.example/v1/aas/auth/login \\
  -H "Content-Type: application/json" \\
  -d '{"email":"admin@aas.com","password":"admin123"}'

curl https://dash.example/v1/aas/analytics/overview?from=...&to=... \\
  -H "Authorization: Bearer $DASH_TOKEN"`;

export function DocsPage() {
  return (
    <Layout>
      <div className="page">
        <h1 className="page-title">API Docs</h1>
        <p className="page-sub">Everything you need to integrate AaaS.</p>

        <div className="panel">
          <h2>Proxy endpoint (send your prompts here)</h2>
          <p>
            <code>POST /v1/chat/completions</code> — OpenAI-compatible. Required headers:
          </p>
          <ul>
            <li>
              <code>X-LLM-API-Key</code> — your own provider key (BYOK)
            </li>
            <li>
              <code>X-LLM-Base-URL</code> — provider base URL (e.g. <code>https://api.openai.com/v1</code>)
            </li>
            <li>
              <code>X-LLM-Format</code> — <code>openai</code> | <code>anthropic</code> | <code>openai-responses</code>
            </li>
            <li>
              <code>X-Tenant-ID</code> — tenant for rate limiting &amp; analytics
            </li>
          </ul>
          <pre className="code-block">{PROXY}</pre>
        </div>

        <div className="panel">
          <h2>Dashboard API</h2>
          <ul>
            <li>
              <code>POST /v1/aas/auth/login</code> — obtain a JWT
            </li>
            <li>
              <code>GET /v1/aas/analytics/overview</code> — aggregate metrics
            </li>
            <li>
              <code>GET /v1/aas/analytics/usage</code> — hourly time series
            </li>
            <li>
              <code>GET /v1/aas/tenants</code> — tenant list
            </li>
            <li>
              <code>GET /v1/aas/usage-logs</code> — recent requests
            </li>
          </ul>
          <pre className="code-block">{DASH}</pre>
        </div>
      </div>
    </Layout>
  );
}
