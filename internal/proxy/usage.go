package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"aaas/internal/config"
	"aaas/internal/detect"
)

// usageReport is the JSON payload posted to the dashboard backend's
// /v1/aas/usage-logs ingest endpoint.
type usageReport struct {
	TenantID         string            `json:"tenantId"`
	SessionID        string            `json:"sessionId"`
	Model            string            `json:"model,omitempty"`
	Provider         string            `json:"provider,omitempty"`
	Format           string            `json:"format,omitempty"`
	PromptTokens     int               `json:"promptTokens,omitempty"`
	CompletionTokens int               `json:"completionTokens,omitempty"`
	EntitiesDetected int               `json:"entitiesDetected,omitempty"`
	Categories       map[string]int    `json:"categories,omitempty"`
	Action           string            `json:"action"`
	LatencyMs        int               `json:"latencyMs,omitempty"`
}

// usageReporter asynchronously ships per-request analytics to the dashboard
// backend. It is best-effort and never blocks the proxy path.
type usageReporter struct {
	url    string
	client *http.Client
}

func newUsageReporter(cfg config.DashboardConfig) *usageReporter {
	if !cfg.Enabled || cfg.UsageURL == "" {
		return nil
	}
	return &usageReporter{
		url:    cfg.UsageURL,
		client: &http.Client{Timeout: 2 * time.Second},
	}
}

func (u *usageReporter) send(r usageReport) {
	if u == nil {
		return
	}
	body, err := json.Marshal(r)
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, u.url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	// Fire-and-forget; ignore the response.
	go func() {
		resp, err := u.client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
}

func providerOf(baseURL string) string {
	if baseURL == "" {
		return ""
	}
	// crude host extraction without net/url import noise
	for i := len(baseURL) - 1; i >= 0; i-- {
		if baseURL[i] == '/' {
			host := baseURL[i+1:]
			return host
		}
	}
	return baseURL
}

func categoriesOf(entities []detect.Entity) map[string]int {
	m := map[string]int{}
	for _, e := range entities {
		m[e.Type]++
	}
	return m
}
