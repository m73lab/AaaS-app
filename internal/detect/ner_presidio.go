package detect

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

// presidioEntity is a single result from the Presidio Analyzer API.
type presidioEntity struct {
	EntityType string  `json:"entity_type"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Score      float64 `json:"score"`
}

// PresidioNER calls a Microsoft Presidio Analyzer service over HTTP(S). Presidio
// runs ONNX-backed NER models (spaCy/transformers) and returns typed entities
// (PERSON, EMAIL_ADDRESS, CREDIT_CARD, PHONE_NUMBER, LOCATION, ORGANIZATION,
// ...). This is the production NER path: high recall for unstructured PII that
// regex/dictionaries miss, without pulling CGO/ONNX native libs into this
// process. Run Presidio behind the mTLS edge (deploy/homelab02) so the link is
// mutually authenticated.
type PresidioNER struct {
	name   string
	url    string
	client *http.Client
	lang   string
}

// NewPresidioNER returns a Presidio-backed detector. When url is empty it is a
// no-op. When caPath/clientCert/clientKey are provided, the HTTP client uses
// mutual TLS: it verifies the Presidio server cert (caPath) and presents its
// own client certificate.
func NewPresidioNER(url, lang, caPath, clientCert, clientKey string) Detector {
	if url == "" {
		return &nerStub{name: "presidio:disabled"}
	}
	if lang == "" {
		lang = "es"
	}
	client := &http.Client{Timeout: 5 * time.Second}
	if caPath != "" && clientCert != "" && clientKey != "" {
		caPEM, err := os.ReadFile(caPath)
		if err == nil {
			pool := x509.NewCertPool()
			if pool.AppendCertsFromPEM(caPEM) {
				cert, cerr := tls.LoadX509KeyPair(clientCert, clientKey)
				if cerr == nil {
					client = &http.Client{
						Timeout: 5 * time.Second,
						Transport: &http.Transport{
							TLSClientConfig: &tls.Config{
								RootCAs:      pool,
								Certificates: []tls.Certificate{cert},
							},
						},
					}
				}
			}
		}
	}
	return &PresidioNER{name: "presidio", url: url, client: client, lang: lang}
}

func (p *PresidioNER) Name() string { return p.name }

func (p *PresidioNER) Detect(text string) ([]Entity, error) {
	body, _ := json.Marshal(map[string]any{
		"text":     text,
		"language": p.lang,
	})
	req, err := http.NewRequest(http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errFromStatus(resp.StatusCode)
	}
	var raw []presidioEntity
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]Entity, 0, len(raw))
	for _, e := range raw {
		out = append(out, Entity{
			Type:  normalizePresidioType(e.EntityType),
			Text:  text[e.Start:e.End],
			Start: e.Start,
			End:   e.End,
			Score: e.Score,
		})
	}
	return out, nil
}

// normalizePresidioType maps Presidio entity types to our uppercase categories.
func normalizePresidioType(t string) string {
	switch t {
	case "EMAIL_ADDRESS":
		return "EMAIL"
	case "PHONE_NUMBER":
		return "PHONE"
	case "CREDIT_CARD":
		return "CREDIT_CARD"
	case "PERSON":
		return "PERSON"
	case "LOCATION":
		return "LOCATION"
	case "ORGANIZATION":
		return "ORG"
	default:
		return t // DATE_TIME, IP_ADDRESS, URL, NRP, etc. pass through
	}
}

type statusError struct{ code int }

func (e *statusError) Error() string { return "presidio analyzer returned non-200" }
func errFromStatus(code int) error   { return &statusError{code: code} }
