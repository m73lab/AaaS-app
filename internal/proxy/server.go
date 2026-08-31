package proxy

import (
	"bufio"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"aaas/internal/audit"
	"aaas/internal/config"
	"aaas/internal/detect"
	"aaas/internal/transform"
	"aaas/internal/vault"
)

// newSessionID returns a random hex session identifier used to scope the
// reversible-token vault for a single request/response round trip.
func newSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Server is the AaaS LLM proxy gateway.
type Server struct {
	engine     *transform.Engine
	upstream   *Upstream
	addr       string
	auditor    *audit.Auditor
	failClosed bool
	tenantHdr  string
	kms        vault.KMS
	http       *http.Server

	limiter  *rateLimiter
	reporter *usageReporter

	reqCount int64
}

// NewServer builds a proxy server. kms is optional and enables key rotation
// via the /v1/admin/rotate-kek endpoint when it implements vault.Rotator.
func NewServer(addr string, engine *transform.Engine, upstream *Upstream, auditor *audit.Auditor, failClosed bool, kms vault.KMS, rlCfg config.RateLimitConfig, dashCfg config.DashboardConfig) *Server {
	s := &Server{
		engine:     engine,
		upstream:   upstream,
		addr:       addr,
		auditor:    auditor,
		failClosed: failClosed,
		tenantHdr:  "X-Tenant-ID",
		kms:        kms,
	}
	if rlCfg.Enabled {
		s.limiter = newRateLimiter(rlCfg)
	}
	s.reporter = newUsageReporter(dashCfg)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/v1/chat/completions", s.handleChat)
	mux.HandleFunc("/v1/anonymize", s.handleAnonymize)
	mux.HandleFunc("/v1/admin/rotate-kek", s.handleRotate)
	s.http = &http.Server{Addr: addr, Handler: mux, ReadTimeout: 60 * time.Second, WriteTimeout: 120 * time.Second}
	return s
}

// Start begins listening (blocking).
func (s *Server) Start() error {
	mode := "forward"
	if s.failClosed {
		mode = "fail-closed"
	}
	log.Printf("AaaS proxy listening on %s (mock mode: %v, policy: %s)", s.addr, s.upstream.APIKey == "", mode)
	return s.http.ListenAndServe()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "aaas_requests_total %d\n", atomic.LoadInt64(&s.reqCount))
	for cat, n := range s.engine.Counts() {
		fmt.Fprintf(w, "aaas_entities_anonymized_total{category=\"%s\"} %d\n", cat, n)
	}
}

func (s *Server) tenantOf(r *http.Request) string {
	t := r.Header.Get(s.tenantHdr)
	if t == "" {
		return "default"
	}
	return t
}

func aggregate(entities []detect.Entity) map[string]int {
	m := map[string]int{}
	for _, e := range entities {
		m[e.Type]++
	}
	return m
}

// handleChat is the OpenAI-compatible anonymizing proxy.
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	atomic.AddInt64(&s.reqCount, 1)
	start := time.Now()

	// --- Required headers (BYOK) ------------------------------------------
	clientKey := r.Header.Get("X-LLM-API-Key")
	clientBaseURL := r.Header.Get("X-LLM-Base-URL")
	tenant := s.tenantOf(r)
	if clientKey == "" || clientBaseURL == "" {
		http.Error(w, "missing required headers: X-LLM-API-Key and X-LLM-Base-URL are mandatory", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	session := newSessionID()
	format := LLMFormat(r.Header.Get("X-LLM-Format"))

	// Determine streaming flag and model from the (unmodified) body.
	stream := false
	model := ""
	var generic map[string]json.RawMessage
	if json.Unmarshal(body, &generic) == nil {
		if raw, ok := generic["stream"]; ok {
			_ = json.Unmarshal(raw, &stream)
		}
		if raw, ok := generic["model"]; ok {
			_ = json.Unmarshal(raw, &model)
		}
	}

	// --- Per-tenant rate limiting ---------------------------------------
	if s.limiter != nil {
		ok, minCount, hourCount := s.limiter.allow(tenant, start)
		if !ok {
			w.Header().Set("Retry-After", "60")
			w.Header().Set("X-RateLimit-Minute", itoa(minCount))
			w.Header().Set("X-RateLimit-Hour", itoa(hourCount))
			http.Error(w, "rate limit exceeded for tenant", http.StatusTooManyRequests)
			s.reportUsage(session, tenant, model, clientBaseURL, string(format), nil, "rate_limited", start)
			return
		}
	}

	// Anonymize every string in the request body (recursive).
	modified, entities, aerr := anonymizeBody(body, s.engine, session)

	// Fail-closed: detection error or (optionally) an error while anonymizing.
	if aerr != nil {
		if s.failClosed {
			s.audit(audit.Record{
				Session: session, Tenant: tenant, Action: "error",
				Error: aerr.Error(),
			})
			http.Error(w, "anonymization failed (fail-closed)", http.StatusBadGateway)
			return
		}
		// Degrade to forwarding the original payload (logged).
		modified = body
	}

	// Block: any entity mapped to the "block" action must never egress.
	if s.engine.IsBlocked(entities) {
		s.audit(audit.Record{
			Session: session, Tenant: tenant, Action: "blocked",
			Entities: len(entities), Categories: aggregate(entities),
		})
		s.reportUsage(session, tenant, model, clientBaseURL, string(format), entities, "blocked", start)
		http.Error(w, "request blocked: contains PII category forbidden by policy", http.StatusUnprocessableEntity)
		return
	}

	resp, err := s.upstream.Chat(r.Context(), modified, stream, clientKey, clientBaseURL, format)
	if err != nil {
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if stream {
		s.streamResponse(w, r, resp, session)
	} else {
		out, _ := io.ReadAll(resp.Body)
		out = deanonJSON(out, session, s.engine)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(out)
	}

	s.audit(audit.Record{
		Session: session, Tenant: tenant, Action: "forward",
		Entities: len(entities), Categories: aggregate(entities),
	})
	s.reportUsage(session, tenant, model, clientBaseURL, string(format), entities, "forward", start)
}

func (s *Server) reportUsage(session, tenant, model, baseURL, format string, entities []detect.Entity, action string, start time.Time) {
	if s.reporter == nil {
		return
	}
	s.reporter.send(usageReport{
		TenantID:         tenant,
		SessionID:        session,
		Model:            model,
		Provider:         providerOf(baseURL),
		Format:           format,
		EntitiesDetected: len(entities),
		Categories:       categoriesOf(entities),
		Action:           action,
		LatencyMs:        int(time.Since(start).Milliseconds()),
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// handleAnonymize is a generic endpoint for RAG ingestion / batch payloads:
// it anonymizes an arbitrary JSON body and returns the anonymized form plus
// the detected entity metadata (PII-free).
func (s *Server) handleAnonymize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	session := newSessionID()
	modified, entities, aerr := anonymizeBody(body, s.engine, session)
	if aerr != nil {
		if s.failClosed {
			http.Error(w, "anonymization failed (fail-closed)", http.StatusBadGateway)
			return
		}
		modified = body
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"anonymized": string(modified),
		"entities":   entities,
		"blocked":    s.engine.IsBlocked(entities),
	})
}

// streamResponse proxies an SSE stream, de-anonymizing content/arguments on the
// fly and flushing any trailing defragmented token at the end.
func (s *Server) streamResponse(w http.ResponseWriter, r *http.Request, resp *http.Response, session string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	reader := bufio.NewReader(resp.Body)
	st := newStreamState(session, s.engine)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if data == "[DONE]" {
					fmt.Fprintf(w, "data: [DONE]\n\n")
					flusher.Flush()
					break
				}
				processed := deanonStreamJSON([]byte(data), st)
				if processed != nil {
					fmt.Fprintf(w, "data: %s\n\n", processed)
					flusher.Flush()
				}
			} else if line != "" {
				fmt.Fprint(w, line+"\n")
				flusher.Flush()
			}
		}
		if err != nil {
			break
		}
	}
	if tail := st.process(""); tail != "" {
		obj := map[string]any{"choices": []map[string]any{{"index": 0, "delta": map[string]any{"content": tail}, "finish_reason": "stop"}}}
		if b, err := json.Marshal(obj); err == nil {
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}

func (s *Server) audit(rec audit.Record) {
	if s.auditor != nil {
		s.auditor.Log(rec)
	}
}

// handleRotate promotes a new KEK version in the KMS and re-seals every stored
// vault value under it. This is a privileged admin operation. It is protected
// two ways (defense in depth):
//  1. Network layer: deployed behind the mTLS edge (nginx) requiring a client
//     certificate (see deploy/homelab02/nginx/edge.conf).
//  2. Application layer: an admin key in X-AaaS-Admin-Key (constant-time
//     compare) when AaaS_ADMIN_KEY is set.
func (s *Server) handleRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if key := os.Getenv("AaaS_ADMIN_KEY"); key != "" {
		got := r.Header.Get("X-AaaS-Admin-Key")
		if subtle.ConstantTimeCompare([]byte(got), []byte(key)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}
	if s.kms == nil {
		http.Error(w, "KMS not configured", http.StatusBadRequest)
		return
	}
	rot, ok := s.kms.(vault.Rotator)
	if !ok {
		http.Error(w, "KMS does not support rotation", http.StatusBadRequest)
		return
	}
	newVersion := rot.Rotate()
	if err := s.engine.VaultRotate(r.Context()); err != nil {
		http.Error(w, "vault rotate failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"new_version": newVersion})
}
