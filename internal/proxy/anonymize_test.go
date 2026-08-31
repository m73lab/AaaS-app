package proxy

import (
	"context"
	"strings"
	"testing"
	"time"

	"aaas/internal/detect"
	"aaas/internal/policy"
	"aaas/internal/transform"
	"aaas/internal/vault"
)

func testEngine() *transform.Engine {
	det := detect.NewCombined(detect.StructuredDetectors(true)...)
	v := vault.NewMemory(time.Minute)
	p := &policy.Policy{Default: policy.ActionReversible}
	return transform.New(det, v, p, []byte("0123456789abcdef0123456789abcdef"), time.Minute)
}

func TestAnonymizeBodyToolCalls(t *testing.T) {
	engine := testEngine()
	body := []byte(`{
		"messages":[{"role":"user","content":"mail juan@ejemplo.com"}],
		"tools":[{"function":{"arguments":"{\"name\":\"Juan Perez\",\"phone\":\"+56912345678\"}"}}]
	}`)
	modified, entities, err := anonymizeBody(body, engine, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(modified), "juan@ejemplo.com") {
		t.Fatalf("email not anonymized: %s", modified)
	}
	if strings.Contains(string(modified), "+56912345678") {
		t.Fatalf("phone in tool args not anonymized: %s", modified)
	}
	if len(entities) < 2 {
		t.Fatalf("expected >=2 entities, got %d (%v)", len(entities), entities)
	}
}

func TestDeanonBodyRestoresReversible(t *testing.T) {
	engine := testEngine()
	// Simulate an upstream reply whose content carries a reversible token.
	resp := []byte(`{"choices":[{"message":{"content":"contacta a <EMAIL_1> o <PHONE_1>"}}]}`)
	// First register the mappings by anonymizing the original request.
	_, _ = engine.Anonymize(context.Background(), "s2", "juan@ejemplo.com")
	_ = engine // mappings stored in vault under session s2
	// Inject known mappings for s2 via direct vault put is internal; instead
	// anonymize a request in s2 then de-anon a response in s2.
	req := []byte(`{"messages":[{"role":"user","content":"juan@ejemplo.com y +56912345678"}]}`)
	_, _, _ = anonymizeBody(req, engine, "s2")
	deanon := deanonBody(resp, engine, "s2")
	if !strings.Contains(string(deanon), "juan@ejemplo.com") {
		t.Fatalf("reversible token not restored: %s", deanon)
	}
	if !strings.Contains(string(deanon), "+56912345678") {
		t.Fatalf("phone token not restored: %s", deanon)
	}
}

func TestStreamDefrag(t *testing.T) {
	engine := testEngine()
	st := newStreamState("s3", engine)
	// Register mapping for EMAIL_1 in session s3.
	_, _ = engine.Anonymize(context.Background(), "s3", "juan@ejemplo.com")
	// Token split across two fragments.
	st.process("foo <EMA")
	p2 := st.process("IL_1> bar")
	if !strings.Contains(p2, "juan@ejemplo.com") {
		t.Fatalf("split token not reassembled: %q", p2)
	}
}
