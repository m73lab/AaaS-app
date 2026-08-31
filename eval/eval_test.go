package eval

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

func TestEnvelopeEncryption(t *testing.T) {
	kek := make([]byte, 32)
	for i := range kek {
		kek[i] = byte(i)
	}
	kms, err := vault.NewLocalKMS(kek)
	if err != nil {
		t.Fatal(err)
	}
	pt := []byte("secret-PII-value-12345")
	env, err := vault.Seal(context.Background(), kms, pt)
	if err != nil {
		t.Fatal(err)
	}
	// Stored form must not contain the plaintext.
	b, _ := vault.MarshalEnvelope(env)
	if strings.Contains(string(b), "secret-PII") {
		t.Fatal("envelope leaks plaintext")
	}
	got, err := vault.Open(context.Background(), kms, env)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(pt) {
		t.Fatalf("envelope round-trip mismatch: %q", got)
	}
}

func TestHeuristicNER(t *testing.T) {
	det := detect.NewHeuristicNER(true, "PERSON", 0.5)
	entities, err := det.Detect("Hola, soy Juan Perez y trabajo en Acme Corp")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, e := range entities {
		found[e.Type] = true
	}
	if !found["PERSON"] {
		t.Fatalf("heuristic NER missed proper noun (found: %v)", entities)
	}
}

func TestIsBlocked(t *testing.T) {
	p := &policy.Policy{
		Default: policy.ActionReversible,
		Rules:   []policy.Rule{{Categories: []string{"CREDIT_CARD"}, Action: policy.ActionBlock}},
	}
	engine := transform.New(
		detect.NewCombined(detect.StructuredDetectors(false)...),
		vault.NewMemory(time.Minute),
		p, nil, time.Minute,
	)
	entities, _ := engine.Analyze(context.Background(), "tarjeta 4111 1111 1111 1111")
	if !engine.IsBlocked(entities) {
		t.Fatal("expected CREDIT_CARD to be blocked")
	}
}

// sample defines a test prompt and the PII categories that MUST be detected.
type sample struct {
	text    string
	expect  []string
}

var samples = []sample{
	{
		text:   "Hola, soy Juan Pérez y mi correo es juan.perez@example.com",
		expect: []string{"PERSON", "EMAIL"},
	},
	{
		text:   "Mi RUT es 12.345.678-5 y la tarjeta 4111 1111 1111 1111",
		expect: []string{"RUT", "CREDIT_CARD"},
	},
	{
		text:   "Llámame al +56912345678 o escríbeme a soporte@acme.cl",
		expect: []string{"PHONE", "EMAIL"},
	},
	{
		text:   "Servidor 10.0.0.15 pertenece a Acme Corp",
		expect: []string{"IPV4", "CUSTOM"},
	},
}

func buildDetector() detect.Detector {
	detectors := []detect.Detector{}
	detectors = append(detectors, detect.StructuredDetectors(true)...)
	detectors = append(detectors,
		detect.NewDictionaryDetector("PERSON", []string{"Juan Pérez", "María González"}, 0.8),
		detect.NewDictionaryDetector("CUSTOM", []string{"Acme Corp"}, 0.85),
		detect.NewHeuristicNER(false, "PERSON", 0.5),
	)
	return detect.NewCombined(detectors...)
}

func TestDetectionRecall(t *testing.T) {
	det := buildDetector()
	for _, s := range samples {
		entities, err := det.Detect(s.text)
		if err != nil {
			t.Fatalf("detect error: %v", err)
		}
		found := map[string]bool{}
		var foundText []string
		for _, e := range entities {
			found[e.Type] = true
			foundText = append(foundText, e.Type+":"+e.Text)
		}
		for _, cat := range s.expect {
			if !found[cat] {
				t.Errorf("missed expected category %q in: %q (found: %v)", cat, s.text, foundText)
			}
		}
	}
}

func TestFalsePositiveLuhn(t *testing.T) {
	det := detect.NewCombined(detect.StructuredDetectors(false)...)
	entities, _ := det.Detect("El número 1234 5678 9012 3456 no es tarjeta")
	for _, e := range entities {
		if e.Type == "CREDIT_CARD" {
			t.Errorf("non-Luhn number incorrectly flagged as CREDIT_CARD: %q", e.Text)
		}
	}
}

func TestReversibleRoundTrip(t *testing.T) {
	det := buildDetector()
	v := vault.NewMemory(time.Minute)
	defer v.Close()
	p := &policy.Policy{Default: policy.ActionReversible}
	engine := transform.New(det, v, p, nil, time.Minute)

	session := "sess-1"
	in := "Mi correo es juan@ejemplo.com"
	out, err := engine.Anonymize(context.Background(), session, in)
	if err != nil {
		t.Fatalf("anonymize error: %v", err)
	}
	if strings.Contains(out, "juan@ejemplo.com") {
		t.Fatalf("PII leaked through anonymization: %s", out)
	}
	if !strings.Contains(out, "<EMAIL_1>") {
		t.Fatalf("expected reversible token not present: %s", out)
	}
	back := engine.DeAnonymize(context.Background(), session, out)
	if back != in {
		t.Fatalf("round-trip mismatch: got %q want %q", back, in)
	}
}

func TestFPEProperties(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	a := transform.FPE("4111111111111111", "CREDIT_CARD", key)
	b := transform.FPE("4111111111111111", "CREDIT_CARD", key)
	if a != b {
		t.Fatalf("FPE not deterministic: %q vs %q", a, b)
	}
	if a == "4111111111111111" {
		t.Fatal("FPE returned original value (not anonymized)")
	}
	// format preservation: same length and all digits
	if len(a) != 16 {
		t.Fatalf("FPE changed length: %q", a)
	}
	for _, r := range a {
		if r < '0' || r > '9' {
			t.Fatalf("FPE broke digit class: %q", a)
		}
	}
}

func TestFPEIrreversibleNoVault(t *testing.T) {
	det := buildDetector()
	// Null vault: FPE must still work (vault-less).
	v := vault.NewMemory(time.Minute)
	defer v.Close()
	p := &policy.Policy{Default: policy.ActionFPE}
	engine := transform.New(det, v, p, []byte("key-1234567890abcdef0123456789ab"), time.Minute)
	session := "sess-fpe"
	in := "Hola María González"
	out, err := engine.Anonymize(context.Background(), session, in)
	if err != nil {
		t.Fatalf("anonymize error: %v", err)
	}
	if strings.Contains(out, "María González") {
		t.Fatalf("FPE leaked original: %s", out)
	}
	// De-anonymization cannot recover FPE values.
	back := engine.DeAnonymize(context.Background(), session, out)
	if back != out {
		t.Fatalf("FPE should not be reversible: %s -> %s", out, back)
	}
}
