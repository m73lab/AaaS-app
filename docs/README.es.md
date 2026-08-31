# AaaS — Anonymization as a Service

> **Una capa de gobernanza PII para sistemas de IA.** Detecta, transforma y restaura datos sensibles en tráfico LLM — sin romper el prompt.

[English](../README.md) | [Español](README.es.md)

---

## Qué Hace

AaaS se interpone entre tus aplicaciones/agentes y los proveedores LLM (OpenAI, Anthropic, etc.), actuando como un **proxy con conciencia de privacidad**:

```
Tu App/Agente
      │
      ▼
┌─────────────────────────────────────┐
│           AaaS Proxy                │
│  detectar → transformar → enviar →  │
│  desanonimizar → responder          │
└─────────────────────────────────────┘
      │              │
      ▼              ▼
  Proveedor LLM   Vault / Redis
```

1. **Intercepta** prompts y completions vía un endpoint API compatible
2. **Detecta** PII usando regex, diccionarios, NER heurístico y Microsoft Presidio
3. **Transforma** datos sensibles (tokenización, FPE, redacción o bloqueo)
4. **Envía** el prompt limpio al LLM upstream
5. **Restaura** los valores originales en la respuesta antes de devolverla al cliente

---

## Características

### Transformación Dual
- **Reversible** (`reversible`): token almacenado en vault, restaurado en la respuesta
- **FPE sin vault** (`fpe`): cifrado determinístico que preserva el formato — sin almacén central de PII
- **Redactar** / **Hashear** / **Bloquear** por categoría

### Conjunto de Detección
- **Regex**: emails, tarjetas de crédito (con Luhn), RUT chileno, teléfonos, IPv4
- **Diccionario**: términos sensibles personalizados
- **Presidio NER** (ONNX): reconocimiento de entidades de Microsoft vía sidecar
- **NER Heurístico**: secuencias de palabras capitalizadas con filtro de stopwords

### Política como Código
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

### Seguridad
- **Cifrado envelope** con KMS rotativo (AES-GCM, rotación de KEK en vivo)
- **mTLS** a Redis y Presidio
- **Log de auditoría sin PII** (solo metadata, JSON-lines)
- **Modo fail-closed** + reglas de bloqueo para categorías de alto riesgo
- **Rate limiting** por tenant (ventana deslizante)

### Soporte de Streaming
- Streaming SSE completo con defragmentación de tokens
- Maneja `tool_calls` / argumentos de `function`
- Payloads RAG arbitrarios vía `/v1/anonymize`

---

## Arquitectura

```
┌─────────────┐     ┌──────────────────────────┐     ┌─────────────┐
│  App / Agente│────▶│     AaaS Proxy (Go)       │────▶│ Proveedor   │
│              │◀────│  detectar → transformar → │◀────│ LLM         │
└─────────────┘     │  enviar → desanonimizar    │     └─────────────┘
                    └──────────┬─────────────────┘
                               │
                 ┌─────────────┼─────────────┐
                 ▼             ▼             ▼
           ┌──────────┐ ┌──────────┐ ┌──────────────┐
           │  Vault   │ │ Presidio │ │ Dashboard    │
           │ (Redis/  │ │  (NER)   │ │ (React+API)  │
           │  Memoria)│ │          │ │              │
           └──────────┘ └──────────┘ └──────────────┘
```

| Componente | Tecnología | Propósito |
|-----------|-----------|-----------|
| **Proxy** | Go 1.27 | Detección, transformación y reenvío de PII |
| **Vault** | Redis (TLS) o en memoria | Almacenamiento de tokens cifrados con envelope encryption |
| **Presidio** | Python / ONNX NER | Sidecar de reconocimiento de entidades |
| **Dashboard Backend** | Node.js / Express | Gestión de tenants, analíticas, API admin |
| **Dashboard Frontend** | React 19 / Vite | UI admin para tenants, logs, analíticas |

---

## Inicio Rápido

### Opción A: Docker Compose (recomendado)

```bash
# Clonar el repo
git clone https://github.com/m73lab/AaaS-app.git
cd AaaS-app

# Generar certificados TLS
make gen-certs

# Configurar variables de entorno
export AaaS_MASTER_KEY="tu-clave-maestra-32-bytes"
export AaaS_ADMIN_KEY="tu-clave-admin"
export AaaS_KEK_VERSION=1

# Desplegar el stack completo
docker compose -f deploy/homelab02/docker-compose.yml up -d

# Verificar
curl http://localhost:8080/healthz
```

### Opción B: Desarrollo Local

```bash
# Ejecutar con vault en memoria (sin Docker)
make run

# O directamente:
go run ./cmd/proxy -config config.yaml
```

### Opción C: Desde un release

```bash
# Descargar el binario más reciente
curl -L https://github.com/m73lab/AaaS-app/releases/latest/download/aaas-proxy-linux-amd64 -o aaas-proxy
chmod +x aaas-proxy

# Ejecutar con tu configuración
./aaas-proxy -config config.yaml
```

---

## Configuración

El proxy lee un archivo YAML. Ver [`config.yaml`](../config.yaml) para el esquema completo.

### Configuración Principal

```yaml
server:
  listen: ":8080"

upstream:
  base_url: "https://api.openai.com/v1"
  api_key: ""            # o usar headers BYOK
  timeout: 30

vault:
  type: "memory"         # "memory" | "redis"
  ttl_seconds: 300
  kms:
    type: "rotating"     # "local" | "rotating"
    master_key_env: "AaaS_MASTER_KEY"
```

### Detección

```yaml
detect:
  enable_phone: true
  heuristic_ner: true
  presidio_url: ""       # vacío = deshabilitado
  presidio_lang: "es"
  person_names: []       # diccionario personalizado
```

### Política

```yaml
policy:
  default_action: "reversible"
  fail_closed: false     # true = bloquear en error de detección
  rules:
    - categories: ["CREDIT_CARD"]
      action: "block"
    - categories: ["EMAIL", "RUT", "PHONE", "IPV4"]
      action: "reversible"
    - categories: ["PERSON", "ORG", "CUSTOM"]
      action: "fpe"
```

### Opciones de Vault

| Tipo | Caso de Uso | Trade-off |
|------|-------------|-----------|
| `memory` | Desarrollo / instancia única | Sin persistencia, rápido |
| `redis` | Producción | Persistente, mTLS, soporta rotación |

---

## Referencia API

### Endpoints del Proxy

| Método | Ruta | Auth | Descripción |
|--------|------|------|-------------|
| `POST` | `/v1/chat/completions` | Headers BYOK | Endpoint proxy principal (compatible OpenAI) |
| `POST` | `/v1/anonymize` | Headers BYOK | Anonimización de texto (payloads arbitrarios) |
| `POST` | `/v1/admin/rotate-kek` | mTLS + admin key | Rotación de claves de cifrado |
| `GET` | `/healthz` | Ninguno | Health check |

### Headers Requeridos (BYOK)

| Header | Descripción |
|--------|-------------|
| `X-LLM-API-Key` | API key del LLM upstream |
| `X-LLM-Base-URL` | URL base del LLM upstream |
| `X-Tenant-ID` | Identificador del tenant (rate limiting y auditoría) |

### API del Dashboard

| Método | Ruta | Auth | Descripción |
|--------|------|------|-------------|
| `POST` | `/v1/aas/auth/login` | Email/password | Login admin (devuelve JWT) |
| `GET` | `/v1/aas/auth/me` | JWT | Info del usuario actual |
| `GET` | `/v1/aas/tenants` | JWT (admin) | Listar tenants |
| `POST` | `/v1/aas/tenants` | JWT (admin) | Crear tenant |
| `GET` | `/v1/aas/analytics/overview` | JWT (admin) | Resumen del dashboard |
| `GET` | `/v1/aas/usage-logs` | JWT (admin) | Logs de uso |

---

## Dashboard

El dashboard admin proporciona:

- **Resumen**: conteo de requests, desglose de entidades, requests bloqueados, percentiles de latencia
- **Gráficos de uso**: volumen horario de requests por tenant y modelo
- **Gestión de tenants**: CRUD de tenants con rate limits y API keys
- **Logs**: uso reciente con tenant, modelo, formato, acción y latencia

```bash
# Acceder al dashboard
open http://localhost:3000

# Credenciales por defecto
Email: admin@aas.com
Password: admin123
```

---

## Guía de Despliegue

### Prerrequisitos

- Docker + Docker Compose v2
- OpenSSL (para generación de certificados)
- Acceso a una API key de proveedor LLM

### Despliegue del Stack Completo

```bash
# 1. Clonar y entrar al directorio
git clone https://github.com/m73lab/AaaS-app.git && cd AaaS-app

# 2. Generar certificados TLS (CA, Redis, Presidio)
make gen-certs

# 3. Configurar secretos
export AaaS_MASTER_KEY=$(openssl rand -hex 32)
export AaaS_ADMIN_KEY=$(openssl rand -hex 32)
export AaaS_KEK_VERSION=1

# 4. Construir e iniciar
docker compose -f deploy/homelab02/docker-compose.yml up -d --build

# 5. Verificar
curl http://localhost:8080/healthz
# → {"status":"ok"}
```

### Servicios

| Servicio | Puerto | Descripción |
|----------|--------|-------------|
| `proxy` | 8080 | Proxy AaaS (punto de entrada principal) |
| `dashboard-frontend` | 3000 | UI admin |
| `dashboard-backend` | 3001 | API admin |
| `redis` | 6379 (interno) | Backend del vault |
| `presidio-analyzer` | 5001 (interno) | Sidecar NER |
| `edge` | 5444 (admin) | Endpoint admin mTLS |

### Variables de Entorno

| Variable | Requerida | Descripción |
|----------|-----------|-------------|
| `AaaS_MASTER_KEY` | Sí | Clave de 32 bytes hex para cifrado envelope |
| `AaaS_KEK_VERSION` | Sí | Versión de la clave de cifrado (entero) |
| `AaaS_ADMIN_KEY` | Sí | Clave admin para rotación de KEK |
| `SUPABASE_URL` | Sí | URL del proyecto Supabase (dashboard backend) |
| `SUPABASE_SERVICE_KEY` | Sí | Service role key de Supabase |
| `JWT_SECRET` | Sí | Secreto para firmar JWT (dashboard) |
| `REDIS_PASSWORD` | Sí | Contraseña de Redis |

---

## Desarrollo

### Prerrequisitos

- Go 1.27+
- Node.js 22+ (para el dashboard)
- Docker (para el sidecar de Presidio)

### Configuración Local

```bash
# Iniciar Presidio localmente
cd deploy/presidio && docker compose up -d

# Ejecutar el proxy
make run

# Ejecutar el dashboard
cd dashboard-backend && npm install && npm run dev
cd dashboard-client && npm install && npm run dev
```

### Ejecutar Tests

```bash
# Tests de Go
go test ./...

# Tests del dashboard backend
cd dashboard-backend && npm test

# Tests del dashboard client
cd dashboard-client && npm test
```

### Estructura del Proyecto

```
├── cmd/proxy/          # Punto de entrada del proxy
├── internal/           # Paquetes Go core
│   ├── audit/          # Logging de auditoría sin PII
│   ├── config/         # Carga de configuración YAML
│   ├── detect/         # Detección de PII (regex, NER, Presidio)
│   ├── policy/         # Políticas de acción
│   ├── proxy/          # Servidor HTTP, streaming, rate limiting
│   ├── transform/      # Motor de anonimización
│   └── vault/          # Almacenamiento de tokens (memoria, Redis)
├── dashboard-backend/  # API admin Node.js
├── dashboard-client/   # UI admin React
├── deploy/             # Docker Compose + configs de despliegue
├── scripts/            # Generación de certs, renderizado de config
└── eval/               # Tests de integración
```

---

## Consideraciones de Seguridad

- **Vault encryption**: Todos los tokens PII están cifrados con envelope encryption AES-GCM. La clave maestra nunca sale del entorno.
- **mTLS**: Las conexiones a Redis y Presidio requieren certificados de cliente.
- **Logging de auditoría**: Cada evento de anonimización se registra con metadata solamente (sin PII).
- **Rate limiting**: Ventana deslizante por tenant (configurable por minuto/hora).
- **Fail-closed**: Modo opcional que bloquea todas las requests si la detección falla.
- **Rotación de KEK**: Rotación en vivo de claves sin downtime vía API admin.

---

## Licencia

Ver [LICENSE](../LICENSE) para detalles.
