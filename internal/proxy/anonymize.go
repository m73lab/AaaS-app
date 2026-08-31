package proxy

import (
	"context"
	"encoding/json"

	"aaas/internal/detect"
	"aaas/internal/transform"
)

// walkAnonymize recursively visits every string in a parsed JSON value,
// applying fn (which anonymizes and may return detection entities or an
// error). This covers message content, tool_call arguments, function messages
// and any nested RAG/ingestion payloads — not just messages[].content.
func walkAnonymize(v interface{}, fn func(string) (string, []detect.Entity, error)) (interface{}, []detect.Entity, error) {
	var ents []detect.Entity
	switch val := v.(type) {
	case map[string]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, sv := range val {
			nv, e, err := walkAnonymize(sv, fn)
			if err != nil {
				return nil, ents, err
			}
			ents = append(ents, e...)
			m[k] = nv
		}
		return m, ents, nil
	case []interface{}:
		arr := make([]interface{}, len(val))
		for i, sv := range val {
			nv, e, err := walkAnonymize(sv, fn)
			if err != nil {
				return nil, ents, err
			}
			ents = append(ents, e...)
			arr[i] = nv
		}
		return arr, ents, nil
	case string:
		s, e, err := fn(val)
		if err != nil {
			return nil, ents, err
		}
		return s, e, nil
	default:
		return v, ents, nil
	}
}

// anonymizeBody parses a request body, anonymizes every string value, and
// returns the re-serialized body plus all detected entities.
func anonymizeBody(body []byte, engine *transform.Engine, session string) ([]byte, []detect.Entity, error) {
	var root interface{}
	if err := json.Unmarshal(body, &root); err != nil {
		return body, nil, err
	}
	ctx := context.Background()
	fn := func(s string) (string, []detect.Entity, error) {
		out, ents, err := engine.AnonymizeEntities(ctx, session, s)
		return out, ents, err
	}
	newRoot, ents, err := walkAnonymize(root, fn)
	if err != nil {
		return body, nil, err
	}
	out, err := json.Marshal(newRoot)
	if err != nil {
		return body, nil, err
	}
	return out, ents, nil
}

// deanonByKey recursively restores reversible tokens only within the string
// fields that can carry model content: "content" and "arguments" (tool calls).
// This safely reverses anonymization without touching other fields.
func deanonByKey(v interface{}, engine *transform.Engine, session string) interface{} {
	ctx := context.Background()
	switch val := v.(type) {
	case map[string]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, sv := range val {
			if k == "content" || k == "arguments" {
				if s, ok := sv.(string); ok {
					m[k] = engine.DeAnonymize(ctx, session, s)
					continue
				}
			}
			m[k] = deanonByKey(sv, engine, session)
		}
		return m
	case []interface{}:
		arr := make([]interface{}, len(val))
		for i, sv := range val {
			arr[i] = deanonByKey(sv, engine, session)
		}
		return arr
	default:
		return v
	}
}

// deanonBody restores reversible tokens in a parsed response body.
func deanonBody(body []byte, engine *transform.Engine, session string) []byte {
	var root interface{}
	if err := json.Unmarshal(body, &root); err != nil {
		return body
	}
	newRoot := deanonByKey(root, engine, session)
	out, err := json.Marshal(newRoot)
	if err != nil {
		return body
	}
	return out
}
