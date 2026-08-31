package detect

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPresidioNER verifies the real NER client against a fake Presidio
// Analyzer endpoint (the same JSON contract the real service speaks). This
// proves the integration end-to-end without needing Docker.
func TestPresidioNER(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected JSON content type")
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		text, _ := body["text"].(string)
		// Return a PERSON entity covering "Juan Perez" (indices must match).
		start := indexOf(text, "Juan Perez")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"entity_type": "PERSON",
				"start":       start,
				"end":         start + len("Juan Perez"),
				"score":       0.91,
			},
		})
	}))
	defer srv.Close()

	ner := NewPresidioNER(srv.URL, "es", "", "", "")
	text := "Hola Juan Perez, ¿cómo estás?"
	entities, err := ner.Detect(text)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entities {
		if e.Type == "PERSON" && e.Text == "Juan Perez" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Presidio NER missed PERSON: %v", entities)
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return 0
}
