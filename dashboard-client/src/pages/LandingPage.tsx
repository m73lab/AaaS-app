import { Link } from 'react-router-dom';

export function LandingPage() {
  return (
    <div className="landing">
      <header className="landing-nav">
        <div className="brand">
          <span className="brand-mark">A</span>
          <span className="brand-name">AaaS</span>
        </div>
        <nav className="landing-links">
          <a href="#features">Features</a>
          <a href="#how">How it works</a>
          <Link to="/login" className="btn-primary">
            Sign in
          </Link>
        </nav>
      </header>

      <section className="hero">
        <h1>
          Ship LLM features without leaking <span className="accent">PII</span>.
        </h1>
        <p className="hero-sub">
          AaaS is a drop-in proxy that anonymizes prompts before they reach the model and
          restores real values in the response. Bring your own key, keep your provider billing.
        </p>
        <div className="hero-cta">
          <Link to="/login" className="btn-primary btn-lg">
            Open Dashboard
          </Link>
          <a href="#how" className="btn-ghost btn-lg">
            See how it works
          </a>
        </div>
      </section>

      <section id="features" className="features">
        <div className="feature">
          <h3>Format-agnostic</h3>
          <p>OpenAI, Anthropic and OpenAI Responses — one endpoint, zero code changes.</p>
        </div>
        <div className="feature">
          <h3>Reversible & FPE</h3>
          <p>Detected entities are pseudonymized and restored per-session. Never train on PII.</p>
        </div>
        <div className="feature">
          <h3>Per-tenant limits</h3>
          <p>Sliding-window rate limiting and live usage analytics per tenant.</p>
        </div>
        <div className="feature">
          <h3>Self-hosted</h3>
          <p>Runs on your homelab with mTLS, Redis and Presidio NER. Your data stays yours.</p>
        </div>
      </section>

      <section id="how" className="how">
        <h2>How it works</h2>
        <ol>
          <li>Your app calls <code>/v1/chat/completions</code> with <code>X-LLM-API-Key</code>, <code>X-LLM-Base-URL</code> and <code>X-Tenant-ID</code>.</li>
          <li>AaaS detects PII (regex, dictionaries, Presidio NER) and replaces it with reversible tokens.</li>
          <li>The anonymized prompt is sent to your LLM provider. The response is de-anonymized on the way back.</li>
          <li>Every request is logged for analytics — with zero raw PII stored.</li>
        </ol>
      </section>

      <footer className="landing-foot">
        <span>AaaS — Anonymization as a Service</span>
      </footer>
    </div>
  );
}
