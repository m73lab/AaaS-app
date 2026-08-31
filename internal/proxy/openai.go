package proxy

// Message is a single chat message. Only the string Content is anonymized in
// the MVP; tool_call / function content is passed through unchanged (tracked
// as a future hardening item).
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
