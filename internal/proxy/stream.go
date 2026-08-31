package proxy

import (
	"context"
	"encoding/json"
	"strings"

	"aaas/internal/transform"
)

// streamState carries per-response defragmentation state so reversible tokens
// split across SSE chunks are reassembled before de-anonymization.
type streamState struct {
	pending string
	engine  *transform.Engine
	session string
}

func newStreamState(session string, engine *transform.Engine) *streamState {
	return &streamState{session: session, engine: engine}
}

// process de-anonymizes a content fragment, holding back any trailing
// incomplete token (an unclosed "<...>") until more data arrives.
func (s *streamState) process(text string) string {
	full := s.pending + text
	lastOpen := strings.LastIndex(full, "<")
	lastClose := strings.LastIndex(full, ">")
	if lastOpen >= 0 && lastOpen > lastClose {
		s.pending = full[lastOpen:]
		full = full[:lastOpen]
	} else {
		s.pending = ""
	}
	if full == "" {
		return ""
	}
	return s.cut(full)
}

// cut de-anonymizes a complete (token-closed) fragment.
func (s *streamState) cut(full string) string {
	return s.engine.DeAnonymize(context.Background(), s.session, full)
}

// deanonJSON restores reversible tokens in a (non-stream or stream) response
// object. It handles both message.content and tool_calls[].function.arguments
// (and the streaming delta equivalents), leaving other fields untouched.
func deanonJSON(raw []byte, session string, engine *transform.Engine) []byte {
	st := newStreamState(session, engine)
	return deanonStreamJSON(raw, st)
}

func deanonStreamJSON(raw []byte, st *streamState) []byte {
	ctx := context.Background()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}
	rawChoices, ok := m["choices"]
	if !ok {
		return raw
	}
	var choices []map[string]json.RawMessage
	if err := json.Unmarshal(rawChoices, &choices); err != nil {
		return raw
	}
	for i, ch := range choices {
		if rd, ok := ch["delta"]; ok {
			var delta map[string]json.RawMessage
			if json.Unmarshal(rd, &delta) == nil {
				if rc, ok := delta["content"]; ok {
					var content string
					if json.Unmarshal(rc, &content) == nil {
						content = st.process(content)
						b, _ := json.Marshal(content)
						delta["content"] = b
					}
				}
				if rt, ok := delta["tool_calls"]; ok {
					delta["tool_calls"] = deanonToolCalls(rt, st.engine, st.session)
				}
				b, _ := json.Marshal(delta)
				choices[i]["delta"] = b
			}
		}
		if rm, ok := ch["message"]; ok {
			var msg map[string]json.RawMessage
			if json.Unmarshal(rm, &msg) == nil {
				if rc, ok := msg["content"]; ok {
					var content string
					if json.Unmarshal(rc, &content) == nil {
						content = st.engine.DeAnonymize(ctx, st.session, content)
						b, _ := json.Marshal(content)
						msg["content"] = b
					}
				}
				if rt, ok := msg["tool_calls"]; ok {
					msg["tool_calls"] = deanonToolCalls(rt, st.engine, st.session)
				}
				b, _ := json.Marshal(msg)
				choices[i]["message"] = b
			}
		}
	}
	b, err := json.Marshal(choices)
	if err != nil {
		return raw
	}
	m["choices"] = b
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return out
}

// deanonToolCalls restores tokens inside each tool call's arguments string.
func deanonToolCalls(raw json.RawMessage, engine *transform.Engine, session string) json.RawMessage {
	ctx := context.Background()
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return raw
	}
	for i, tc := range arr {
		if rf, ok := tc["function"]; ok {
			var fn map[string]json.RawMessage
			if json.Unmarshal(rf, &fn) == nil {
				if ra, ok := fn["arguments"]; ok {
					var args string
					if json.Unmarshal(ra, &args) == nil {
						args = engine.DeAnonymize(ctx, session, args)
						b, _ := json.Marshal(args)
						fn["arguments"] = b
						b2, _ := json.Marshal(fn)
						arr[i]["function"] = b2
					}
				}
			}
		}
	}
	b, _ := json.Marshal(arr)
	return b
}
