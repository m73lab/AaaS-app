package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LLMFormat represents the API format of the target provider.
type LLMFormat string

const (
	FormatOpenAI          LLMFormat = "openai"           // /v1/chat/completions
	FormatAnthropic       LLMFormat = "anthropic"        // /v1/messages
	FormatOpenAIResponses LLMFormat = "openai-responses" // /v1/responses
)

// Upstream forwards anonymized requests to an LLM provider.
type Upstream struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

// NewUpstream builds an upstream client.
func NewUpstream(baseURL, apiKey string, timeout time.Duration) *Upstream {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &Upstream{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client:  &http.Client{Timeout: timeout},
	}
}

// Chat forwards the anonymized request. clientKey is the tenant's BYOK key.
// clientBaseURL overrides the configured upstream URL. format selects the API
// protocol; when empty it is auto-detected from the URL.
func (u *Upstream) Chat(ctx context.Context, body []byte, stream bool, clientKey, clientBaseURL string, format LLMFormat) (*http.Response, error) {
	key := u.APIKey
	if clientKey != "" {
		key = clientKey
	}
	baseURL := u.BaseURL
	if clientBaseURL != "" {
		baseURL = clientBaseURL
	}
	if format == "" {
		format = detectFormat(baseURL)
	}

	if key == "" {
		return u.mockChat(ctx, body, stream)
	}

	switch format {
	case FormatAnthropic:
		return u.chatAnthropic(ctx, body, stream, key, baseURL)
	case FormatOpenAIResponses:
		return u.chatOpenAIResponses(ctx, body, stream, key, baseURL)
	default:
		return u.chatOpenAI(ctx, body, stream, key, baseURL)
	}
}

// chatOpenAI sends to /v1/chat/completions (OpenAI-compatible).
func (u *Upstream) chatOpenAI(ctx context.Context, body []byte, stream bool, key, baseURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	return u.Client.Do(req)
}

// chatAnthropic sends to /v1/messages (Anthropic format). Converts the
// OpenAI-format request to Anthropic format and wraps the response back.
func (u *Upstream) chatAnthropic(ctx context.Context, body []byte, stream bool, key, baseURL string) (*http.Response, error) {
	anthropicBody, err := openAIToAnthropicRequest(body)
	if err != nil {
		return nil, fmt.Errorf("convert to anthropic: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(baseURL, "/")+"/messages", bytes.NewReader(anthropicBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if stream {
		return resp, nil // SSE passthrough; de-anon handles it
	}
	// Wrap non-streaming Anthropic response into OpenAI format.
	return wrapAnthropicResponse(resp)
}

// chatOpenAIResponses sends to /v1/responses (OpenAI Responses API).
func (u *Upstream) chatOpenAIResponses(ctx context.Context, body []byte, stream bool, key, baseURL string) (*http.Response, error) {
	respBody, err := openAIToResponsesRequest(body)
	if err != nil {
		return nil, fmt.Errorf("convert to responses: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(baseURL, "/")+"/responses", bytes.NewReader(respBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, err
	}
	// Wrap Responses API result into Chat Completions format for de-anon.
	return wrapResponsesResponse(resp)
}

// detectFormat guesses the API format from the base URL.
func detectFormat(baseURL string) LLMFormat {
	lower := strings.ToLower(baseURL)
	if strings.Contains(lower, "anthropic") || strings.Contains(lower, "claude") {
		return FormatAnthropic
	}
	if strings.Contains(lower, "/responses") {
		return FormatOpenAIResponses
	}
	return FormatOpenAI
}

// ── Anthropic conversion ────────────────────────────────────────────

func openAIToAnthropicRequest(body []byte) ([]byte, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	// Extract fields.
	model := getString(m, "model")
	var msgs []Message
	if raw, ok := m["messages"]; ok {
		_ = json.Unmarshal(raw, &msgs)
	}
	var maxTokens int
	if raw, ok := m["max_tokens"]; ok {
		_ = json.Unmarshal(raw, &maxTokens)
	}
	if maxTokens == 0 {
		maxTokens = 4096
	}
	// Build Anthropic messages (skip system, put it separately).
	var system string
	var anthropicMsgs []map[string]any
	for _, msg := range msgs {
		if msg.Role == "system" {
			system = msg.Content
			continue
		}
		role := msg.Role
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		anthropicMsgs = append(anthropicMsgs, map[string]any{
			"role":    role,
			"content": msg.Content,
		})
	}
	out := map[string]any{
		"model":      model,
		"messages":   anthropicMsgs,
		"max_tokens": maxTokens,
	}
	if system != "" {
		out["system"] = system
	}
	return json.Marshal(out)
}

func wrapAnthropicResponse(resp *http.Response) (*http.Response, error) {
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return &http.Response{
			StatusCode: resp.StatusCode,
			Body:       io.NopCloser(bytes.NewReader(raw)),
			Header:     http.Header{"Content-Type": {"application/json"}},
		}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	// Extract content from Anthropic response. Some models (Qwen) return
	// multiple blocks: a "thinking" block first, then a "text" block.
	// We must find the first "text" block, not just the first block.
	content := ""
	if arr, ok := m["content"].([]any); ok {
		for _, item := range arr {
			if block, ok := item.(map[string]any); ok {
				if block["type"] == "text" {
					if t, ok := block["text"].(string); ok && t != "" {
						content = t
						break
					}
				}
			}
		}
	}
	model, _ := m["model"].(string)
	// Build OpenAI-compatible response.
	out := map[string]any{
		"id":      "msg-anthropic",
		"object":  "chat.completion",
		"model":   model,
		"choices": []map[string]any{{
			"index": 0,
			"message": map[string]any{
				"role":    "assistant",
				"content": content,
			},
			"finish_reason": "stop",
		}},
		"usage": map[string]any{"prompt_tokens": 0, "completion_tokens": 0},
	}
	b, _ := json.Marshal(out)
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(b)),
		Header:     http.Header{"Content-Type": {"application/json"}},
	}, nil
}

// ── OpenAI Responses API conversion ────────────────────────────────

func openAIToResponsesRequest(body []byte) ([]byte, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	model := getString(m, "model")
	// Build the "input" from messages.
	var msgs []Message
	if raw, ok := m["messages"]; ok {
		_ = json.Unmarshal(raw, &msgs)
	}
	var input string
	for _, msg := range msgs {
		if msg.Role == "user" {
			input = msg.Content
		}
	}
	out := map[string]any{
		"model": model,
		"input": input,
	}
	return json.Marshal(out)
}

func wrapResponsesResponse(resp *http.Response) (*http.Response, error) {
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return &http.Response{
			StatusCode: resp.StatusCode,
			Body:       io.NopCloser(bytes.NewReader(raw)),
			Header:     http.Header{"Content-Type": {"application/json"}},
		}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	// Extract output text from Responses API.
	content := ""
	if output, ok := m["output"].([]any); ok {
		for _, item := range output {
			if block, ok := item.(map[string]any); ok {
				if block["type"] == "message" {
					if contentArr, ok := block["content"].([]any); ok && len(contentArr) > 0 {
						if c, ok := contentArr[0].(map[string]any); ok {
							content, _ = c["text"].(string)
						}
					}
				}
			}
		}
	}
	model, _ := m["model"].(string)
	out := map[string]any{
		"id":      "resp-openai",
		"object":  "chat.completion",
		"model":   model,
		"choices": []map[string]any{{
			"index": 0,
			"message": map[string]any{
				"role":    "assistant",
				"content": content,
			},
			"finish_reason": "stop",
		}},
		"usage": map[string]any{"prompt_tokens": 0, "completion_tokens": 0},
	}
	b, _ := json.Marshal(out)
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(b)),
		Header:     http.Header{"Content-Type": {"application/json"}},
	}, nil
}

// ── Helpers ─────────────────────────────────────────────────────────

func getString(m map[string]json.RawMessage, key string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}

// mockChat builds a fake provider response.
func (u *Upstream) mockChat(ctx context.Context, body []byte, stream bool) (*http.Response, error) {
	content := lastUserContent(body)
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		if stream {
			for _, chunk := range chunkString(content, 8) {
				obj := map[string]any{
					"choices": []map[string]any{{
						"index":         0,
						"delta":         map[string]any{"content": chunk},
						"finish_reason": nil,
					}},
				}
				b, _ := json.Marshal(obj)
				fmt.Fprintf(pw, "data: %s\n\n", b)
			}
			fmt.Fprint(pw, "data: [DONE]\n\n")
			return
		}
		obj := map[string]any{
			"id":      "chatcmpl-mock",
			"object":  "chat.completion",
			"model":   "mock",
			"choices": []map[string]any{{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": "Confirmo los datos recibidos: " + content,
				},
				"finish_reason": "stop",
			}},
			"usage": map[string]any{"prompt_tokens": 0, "completion_tokens": 0},
		}
		b, _ := json.Marshal(obj)
		fmt.Fprintf(pw, "%s", b)
	}()
	header := http.Header{"Content-Type": {"application/json"}}
	if stream {
		header.Set("Content-Type", "text/event-stream")
	}
	return &http.Response{
		StatusCode: 200, Header: header, Body: pr,
		Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1, ContentLength: -1,
	}, nil
}

func lastUserContent(body []byte) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return ""
	}
	raw, ok := m["messages"]
	if !ok {
		return ""
	}
	var msgs []Message
	if err := json.Unmarshal(raw, &msgs); err != nil {
		return ""
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return msgs[i].Content
		}
	}
	return ""
}

func chunkString(s string, n int) []string {
	runes := []rune(s)
	if len(runes) == 0 {
		return []string{""}
	}
	var out []string
	for i := 0; i < len(runes); i += n {
		end := i + n
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[i:end]))
	}
	return out
}
