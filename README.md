# AaaS — Anonymization as a Service

> **A policy-driven PII governance layer for AI systems.** Detect, transform, and restore sensitive data in LLM traffic — without breaking the prompt.

[English](README.md) | [Español](docs/README.es.md)

---

## What It Does

AaaS sits between your applications/agents and LLM providers (OpenAI, Anthropic, etc.), acting as a **privacy-aware proxy**:

```
Your App/Agent
      │
      ▼
┌─────────────────────────────────────┐
│           AaaS Proxy                │
│  detect → transform → forward →     │
│  de-anonymize → respond             │
└─────────────────────────────────────┘
      │              │
      ▼              ▼
  LLM Provider   Vault / Redis
```

1. **Intercepts** prompts and completions via a drop-in API endpoint
2. **Detects** PII using regex, dictionaries, heuristic NER, and Microsoft Presidio
3. **Transforms** sensitive data (tokenization, FPE, redaction, or blocking)
4. **Forwards** the clean prompt to the upstream LLM
5. **Restores** original values in the response before returning to the client

---

## Features

### Dual-Mode Transformation
- **Reversible** (`reversible`): token stored in a vault, restored in the reply
- **Vault-less FPE** (`fpe`): deterministic, format-preserving encryption — no central PII store
- **Redact** / **Hash** / **Block** actions per category

### Detection Ensemble
- **Regex**: emails, credit cards (with Luhn), RUT (Chilean tax ID), phone numbers, IPv4
- **Dictionary**: custom sensitive term matching
- **Presidio NER** (ONNX): Microsoft's named entity recognition via sidecar
- **Heuristic NER**: capitalized word runs with stopword filtering

### Policy as Code
```yaml
policy:
  default_action: "reversible"
  rules:
    - categories: ["CREDIT_CARD"]
      action: "block"
    - categories: ["EMAIL", "RUT", "PHONE"]
      action: "reversible"
    - categories: ["PERSON", "ORG"]
      action: "fpe"
```

### Security
- **Envelope encryption** with rotating KMS (AES-GCM, live KEK rotation)
- **Mutual TLS** to Redis and Presidio
- **PII-free audit log** (metadata only, JSON-lines)
- **Fail-closed** mode + block rules for high-risk categories
- **Rate limiting** per tenant (sliding window)

### Streaming Support
- Full SSE streaming with token defragmentation
- Handles `tool_calls` / `function` arguments
- Arbitrary RAG payloads via `/v1/anonymize`

---

## Architecture

```
┌─────────────┐     ┌──────────────────────────┐     ┌─────────────┐
│  App / Agent │────▶│     AaaS Proxy (Go)       │────▶│ LLM Provider │
│              │◀────│  detect → transform →     │◀────│ (OpenAI etc) │
└─────────────┘     │  forward → de-anonymize    │     └─────────────┘
                    └──────────┬─────────────────┘
                               │
                 ┌─────────────┼─────────────┐
                 ▼             ▼             ▼
           ┌──────────┐ ┌──────────┐ ┌──────────────┐
           │  Vault   │ │ Presidio │ │ Dashboard    │
           │ (Redis/  │ │  (NER)   │ │ (React+API)  │
           │  Memory) │ │          │ │              │
           └──────────┘ └──────────┘ └──────────────┘
```

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Proxy** | Go 1.27 | Core PII detection, transformation, and forwarding |
| **Vault** | Redis (TLS) or in-memory | Encrypted token storage with envelope encryption |
| **Presidio** | Python / ONNX NER | Named entity recognition sidecar |
| **Dashboard Backend** | Node.js / Express | Tenant management, usage analytics, admin API |
| **Dashboard Frontend** | React 19 / Vite | Admin UI for tenants, logs, analytics |

---

## Quick Start

### Option A: Docker Compose (recommended)

```bash
# Clone the repo
git clone https://github.com/m73lab/AaaS-app.git
cd AaaS-app

# Generate TLS certificates
make gen-certs

# Set environment variables
export AaaS_MASTER_KEY="your-32-byte-master-key-here"
export AaaS_ADMIN_KEY="your-admin-key"
export AaaS_KEK_VERSION=1

# Deploy the full stack
docker compose -f deploy/homelab02/docker-compose.yml up -d

# Access the proxy
curl http://localhost:8080/healthz
```

### Option B: Local Development

```bash
# Run with in-memory vault (no Docker required)
make run

# Or directly:
go run ./cmd/proxy -config config.yaml
```

### Option C: From a release

```bash
# Download the latest binary
curl -L https://github.com/m73lab/AaaS-app/releases/latest/download/aaas-proxy-linux-amd64 -o aaas-proxy
chmod +x aaas-proxy

# Run with your config
./aaas-proxy -config config.yaml
```

---

## Configuration

The proxy reads a YAML config file. See [`config.yaml`](config.yaml) for the full schema.

### Core Settings

```yaml
server:
  listen: ":8080"

upstream:
  base_url: "https://api.openai.com/v1"
  api_key: ""            # or use BYOK headers
  timeout: 30

vault:
  type: "memory"         # "memory" | "redis"
  ttl_seconds: 300
  kms:
    type: "rotating"     # "local" | "rotating"
    master_key_env: "AaaS_MASTER_KEY"
```

### Detection

```yaml
detect:
  enable_phone: true
  heuristic_ner: true
  presidio_url: ""       # empty = disabled
  presidio_lang: "es"
  person_names: []       # custom dictionary
```

### Policy

```yaml
policy:
  default_action: "reversible"
  fail_closed: false     # true = block on detection error
  rules:
    - categories: ["CREDIT_CARD"]
      action: "block"
    - categories: ["EMAIL", "RUT", "PHONE", "IPV4"]
      action: "reversible"
    - categories: ["PERSON", "ORG", "CUSTOM"]
      action: "fpe"
```

### Vault Options

| Type | Use Case | Trade-off |
|------|----------|-----------|
| `memory` | Development / single-instance | No persistence, fast |
| `redis` | Production | Persistent, mTLS, supports rotation |

---

## API Reference

### Proxy Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/v1/chat/completions` | BYOK headers | Main proxy endpoint (OpenAI-compatible) |
| `POST` | `/v1/anonymize` | BYOK headers | Text anonymization (arbitrary payloads) |
| `POST` | `/v1/admin/rotate-kek` | mTLS + admin key | Rotate encryption keys |
| `GET` | `/healthz` | None | Health check |

### Required Headers (BYOK)

| Header | Description |
|--------|-------------|
| `X-LLM-API-Key` | Upstream LLM API key |
| `X-LLM-Base-URL` | Upstream LLM base URL |
| `X-Tenant-ID` | Tenant identifier (for rate limiting and audit) |

### Dashboard API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/v1/aas/auth/login` | Email/password | Admin login (returns JWT) |
| `GET` | `/v1/aas/auth/me` | JWT | Current user info |
| `GET` | `/v1/aas/tenants` | JWT (admin) | List tenants |
| `POST` | `/v1/aas/tenants` | JWT (admin) | Create tenant |
| `GET` | `/v1/aas/analytics/overview` | JWT (admin) | Dashboard overview |
| `GET` | `/v1/aas/usage-logs` | JWT (admin) | Usage logs |

---

## Dashboard

The admin dashboard provides:

- **Overview**: request counts, entity breakdown, blocked requests, latency percentiles
- **Usage charts**: hourly request volume by tenant and model
- **Tenant management**: CRUD for tenants with rate limits and API keys
- **Logs**: recent usage with tenant, model, format, action, and latency

```bash
# Access the dashboard
open http://localhost:3000

# Default credentials
Email: admin@aas.com
Password: admin123
```

---

## Deployment Guide

### Prerequisites

- Docker + Docker Compose v2
- OpenSSL (for certificate generation)
- Access to an LLM provider API key

### Full Stack Deployment

```bash
# 1. Clone and enter the directory
git clone https://github.com/m73lab/AaaS-app.git && cd AaaS-app

# 2. Generate TLS certificates (CA, Redis, Presidio)
make gen-certs

# 3. Set secrets
export AaaS_MASTER_KEY=$(openssl rand -hex 32)
export AaaS_ADMIN_KEY=$(openssl rand -hex 32)
export AaaS_KEK_VERSION=1

# 4. Build and start
docker compose -f deploy/homelab02/docker-compose.yml up -d --build

# 5. Verify
curl http://localhost:8080/healthz
# → {"status":"ok"}
```

### Services

| Service | Port | Description |
|---------|------|-------------|
| `proxy` | 8080 | AaaS proxy (main entry point) |
| `dashboard-frontend` | 3000 | Admin UI |
| `dashboard-backend` | 3001 | Admin API |
| `redis` | 6379 (internal) | Vault backend |
| `presidio-analyzer` | 5001 (internal) | NER sidecar |
| `edge` | 5444 (admin) | mTLS admin endpoint |

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `AaaS_MASTER_KEY` | Yes | 32-byte hex key for envelope encryption |
| `AaaS_KEK_VERSION` | Yes | Key encryption key version (integer) |
| `AaaS_ADMIN_KEY` | Yes | Admin API key for KEK rotation |
| `SUPABASE_URL` | Yes | Supabase project URL (dashboard backend) |
| `SUPABASE_SERVICE_KEY` | Yes | Supabase service role key |
| `JWT_SECRET` | Yes | JWT signing secret (dashboard) |
| `REDIS_PASSWORD` | Yes | Redis password |

---

## Development

### Prerequisites

- Go 1.27+
- Node.js 22+ (for dashboard)
- Docker (for Presidio sidecar)

### Local Setup

```bash
# Start Presidio locally
cd deploy/presidio && docker compose up -d

# Run the proxy
make run

# Run the dashboard
cd dashboard-backend && npm install && npm run dev
cd dashboard-client && npm install && npm run dev
```

### Running Tests

```bash
# Go tests
go test ./...

# Dashboard backend
cd dashboard-backend && npm test

# Dashboard client
cd dashboard-client && npm test
```

### Project Structure

```
├── cmd/proxy/          # Proxy entrypoint
├── internal/           # Core Go packages
│   ├── audit/          # PII-free audit logging
│   ├── config/         # YAML config loading
│   ├── detect/         # PII detection (regex, NER, Presidio)
│   ├── policy/         # Action policies
│   ├── proxy/          # HTTP server, streaming, rate limiting
│   ├── transform/      # Anonymization engine
│   └── vault/          # Token storage (memory, Redis)
├── dashboard-backend/  # Node.js admin API
├── dashboard-client/   # React admin UI
├── deploy/             # Docker Compose + deployment configs
├── scripts/            # Cert generation, config rendering
└── eval/               # Integration tests
```

---

## Security Considerations

- **Vault encryption**: All PII tokens are envelope-encrypted with AES-GCM. The master key never leaves the environment.
- **mTLS**: Redis and Presidio connections require client certificates.
- **Audit logging**: Every anonymization event is logged with metadata only (no PII).
- **Rate limiting**: Per-tenant sliding window (configurable per minute/hour).
- **Fail-closed**: Optional mode that blocks all requests if detection fails.
- **KEK rotation**: Live key rotation without downtime via admin API.

---

## License

See [LICENSE](LICENSE) for details.
