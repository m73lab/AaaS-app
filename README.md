# AaaS — Anonymization as a Service (LLM Proxy)

A low-latency, policy-driven PII governance layer for AI systems. It sits
between your apps/agents and LLM providers (OpenAI, Anthropic, …), detects PII
in requests, transforms it (reversible vault tokenization **or** vault-less
format-preserving encryption), forwards the clean prompt, and de-anonymizes the
response on the way back — without breaking the prompt's structure.

## Features

- **Dual-mode transformation**
  - *Reversible* (`reversible`): token stored in a vault, restored in the reply.
  - *Vault-less FPE* (`fpe`): deterministic, format-preserving, GDPR-friendly
    (no central PII store). Also `redact` / `hash`.
- **Detection ensemble**: regex (email, card w/ Luhn, RUT, phone, IP) +
  custom dictionaries + Microsoft **Presidio** (ONNX NER) sidecar + interim
  heuristic NER.
- **Policy as code**: per-category actions in YAML.
- **Zero-PII audit log** (metadata only) and **fail-closed** + `block` rules.
- **Envelope encryption** with a **rotating KMS** (live KEK rotation, no
  downtime) and **mutual TLS** to Redis and Presidio.
- **Streaming (SSE)** with token defragmentation; covers `tool_calls`/`function`
  arguments and arbitrary RAG payloads (`/v1/anonymize`).

## Architecture

```
 App/Agent
    │  (1) prompt with PII
    ▼
┌──────────────────────────────────────────────────────────┐
│  AaaS proxy (container on homelab02)                       │
│   /v1/chat/completions  ── detect ── transform ── vault    │
│   /v1/admin/rotate-kek  (mTLS + admin key)                │
└───────┬───────────────────────────┬───────────────────────┘
        │ https (mTLS)               │ tls (mTLS) + envelope
        ▼                            ▼
  Presidio edge (nginx)        Redis (tls-port, client-cert)
        │
        ▼
  Presidio Analyzer (ONNX NER)
```
Only `:8080` (proxy) and `:5444` (admin mTLS) are exposed — on loopback, reached
via SSH tunnel. Redis and Presidio are never on the public network.

## Local development (no Docker)

```bash
go test ./...                 # unit + integration tests
go run ./cmd/proxy -config config.yaml
# mock mode (no API key) echoes the anonymized prompt so you can see the round trip
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"messages":[{"role":"user","content":"Hola, soy Juan Perez, mi correo es juan@ejemplo.com"}]}'
```

## Production deploy on homelab02 (Docker Compose)

```bash
export HOMELAB02=user@homelab02
export REDIS_PASSWORD='a-strong-password'
export AaaS_MASTER_KEY='32-byte-master-key-for-envelope-encryption'   # from KMS/HSM in prod
export AaaS_KEK_VERSION=1
export AaaS_ADMIN_KEY='another-strong-secret'        # for KEK rotation

make deploy        # gen-certs + render config + rsync to /opt/aaas + docker compose up --build
make tunnel         # ssh -L 8080:localhost:8080 -L 5444:localhost:5444 homelab02
```
Then point your LLM client at `http://localhost:8080/v1/chat/completions`.

### mTLS details
- **Redis**: only `tls-port 6379`, `tls-auth-clients yes`; the proxy presents
  `redis-client.crt` and verifies the server via `ca.crt`.
- **Presidio**: the proxy calls the nginx `edge` at `presidio-tls:5443` over
  HTTPS, presenting `redis-client.crt` (clientAuth) and verifying `ca.crt`.
- **Admin rotation**: `edge:5444` requires a client certificate *and* the proxy
  checks `X-AaaS-Admin-Key` (defense in depth).

### KEK rotation (zero downtime)
```bash
curl -s -X POST https://localhost:5444/v1/admin/rotate-kek \
  --cert deploy/certs/redis-client.crt --key deploy/certs/redis-client.key \
  --cacert deploy/certs/ca.crt \
  -H 'X-AaaS-Admin-Key: <AaaS_ADMIN_KEY>'
# => {"new_version":2}
```
Old KEK versions stay retained for decryption; the vault is re-sealed under
the new KEK.

## CI
`.github/workflows/ci.yml` runs `go vet`/`go test`, builds the image, scans it
with **Trivy** (fails on HIGH/CRITICAL), and **signs** it with **Cosign**
(keyless, OIDC).

## Layout
```
cmd/proxy              entrypoint
internal/detect        regex + dictionary + Presidio (mTLS) + heuristic NER
internal/transform     anonymize / de-anonymize, FPE, block policy
internal/vault         memory + Redis (mTLS), KMS, envelope + rotation
internal/audit         PII-free audit log
internal/proxy         OpenAI-compatible server, streaming, /v1/anonymize
deploy/homelab02       compose, redis.conf, nginx edge, systemd unit
scripts/               gen-certs, render-config, setup-homelab02, tunnel
config.*.yaml         dev + server-side template
```

## Security notes
- The audit log never contains PII or tokens — only categories and counts.
- Envelope encryption means a Redis compromise yields only KEK-wrapped data
  keys; the KEK lives in the KMS/HSM, never in Redis.
- Protect `AaaS_MASTER_KEY`, `REDIS_PASSWORD`, `AaaS_ADMIN_KEY` — inject via
  env/`.env`, never commit. `deploy/certs/` and `config.homelab02.yaml` are
  git-ignored.
- For production, replace `LocalKMS`/`RotatingKMS` (master key in env) with a
  cloud/HSM KMS (AWS KMS, Vault Transit) implementing the `vault.KMS` interface,
  and install the Spanish Presidio model for `presidio_lang: "es"`.
