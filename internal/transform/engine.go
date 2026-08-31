package transform

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"aaas/internal/detect"
	"aaas/internal/policy"
	"aaas/internal/vault"
)

// tokenRe matches reversible tokens emitted during anonymization, e.g.
// "<EMAIL_1>" or "<CREDIT_CARD_1>". Categories are uppercase with
// underscores; the trailing "_N" is the per-session counter.
var tokenRe = regexp.MustCompile(`<([A-Z_]+)_(\d+)>`)

// Entity aliases the detection entity type for brevity within this package.
type Entity = detect.Entity

// Engine applies detection + policy + vault to anonymize and (where
// reversible) de-anonymize text.
type Engine struct {
	det    detect.Detector
	vault  vault.Vault
	policy *policy.Policy
	fpeKey []byte
	redact string
	ttl    time.Duration

	mu     sync.Mutex
	counts map[string]*int64
}

// New builds a transform engine.
func New(d detect.Detector, v vault.Vault, p *policy.Policy, fpeKey []byte, ttl time.Duration) *Engine {
	redact := "****"
	if fpeKey == nil {
		fpeKey = make([]byte, 32)
	}
	return &Engine{
		det:    d,
		vault:  v,
		policy: p,
		fpeKey: fpeKey,
		redact: redact,
		ttl:    ttl,
		counts: make(map[string]*int64),
	}
}

// Counts returns a snapshot of how many entities of each category have been
// anonymized (for metrics/observability). It never contains the PII itself.
func (e *Engine) Counts() map[string]int64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make(map[string]int64, len(e.counts))
	for k, v := range e.counts {
		out[k] = atomic.LoadInt64(v)
	}
	return out
}

func (e *Engine) bump(category string) {
	e.mu.Lock()
	c, ok := e.counts[category]
	if !ok {
		var v int64
		c = &v
		e.counts[category] = c
	}
	e.mu.Unlock()
	atomic.AddInt64(c, 1)
}

// Analyze detects sensitive entities in text without transforming it.
func (e *Engine) Analyze(ctx context.Context, text string) ([]Entity, error) {
	return e.det.Detect(text)
}

// Anonymize inspects text, applies the configured policy per entity and
// returns the transformed text. Reversible mappings are stored in the vault
// under the given session. A detection error is propagated so the caller can
// fail closed.
func (e *Engine) Anonymize(ctx context.Context, session, text string) (string, error) {
	entities, err := e.det.Detect(text)
	if err != nil {
		return text, err
	}
	if len(entities) == 0 {
		return text, nil
	}
	// Replace from right to left so byte offsets stay valid.
	sort.SliceStable(entities, func(i, j int) bool {
		return entities[i].Start > entities[j].Start
	})
	result := text
	counters := map[string]int{}
	for _, ent := range entities {
		action := e.policy.ActionFor(ent.Type)
		var repl string
		switch action {
		case policy.ActionReversible:
			counters[ent.Type]++
			tok := fmt.Sprintf("<%s_%d>", strings.ToUpper(ent.Type), counters[ent.Type])
			if err := e.vault.Put(ctx, session, tok, ent.Text, e.ttl); err != nil {
				return text, err
			}
			repl = tok
		case policy.ActionFPE:
			repl = FPE(ent.Text, ent.Type, e.fpeKey)
		case policy.ActionHash:
			repl = deterministicHash(ent.Text, e.fpeKey)
		default:
			repl = e.redact
		}
		e.bump(ent.Type)
		result = result[:ent.Start] + repl + result[ent.End:]
	}
	return result, nil
}

// AnonymizeEntities is like Anonymize but also returns the detected entities
// (used for block/fail-closed decisions and audit metadata).
func (e *Engine) AnonymizeEntities(ctx context.Context, session, text string) (string, []Entity, error) {
	entities, err := e.det.Detect(text)
	if err != nil {
		return text, nil, err
	}
	if len(entities) == 0 {
		return text, nil, nil
	}
	out, err := e.anonymizeWith(entities, ctx, session, text)
	return out, entities, err
}

// anonymizeWith applies transforms for an already-detected entity list.
func (e *Engine) anonymizeWith(entities []Entity, ctx context.Context, session, text string) (string, error) {
	sort.SliceStable(entities, func(i, j int) bool {
		return entities[i].Start > entities[j].Start
	})
	result := text
	counters := map[string]int{}
	for _, ent := range entities {
		action := e.policy.ActionFor(ent.Type)
		var repl string
		switch action {
		case policy.ActionReversible:
			counters[ent.Type]++
			tok := fmt.Sprintf("<%s_%d>", strings.ToUpper(ent.Type), counters[ent.Type])
			if err := e.vault.Put(ctx, session, tok, ent.Text, e.ttl); err != nil {
				return text, err
			}
			repl = tok
		case policy.ActionFPE:
			repl = FPE(ent.Text, ent.Type, e.fpeKey)
		case policy.ActionHash:
			repl = deterministicHash(ent.Text, e.fpeKey)
		default:
			repl = e.redact
		}
		e.bump(ent.Type)
		result = result[:ent.Start] + repl + result[ent.End:]
	}
	return result, nil
}

// IsBlocked reports whether any entity maps to the "block" action, i.e. the
// request must be rejected rather than forwarded (never egress that PII).
func (e *Engine) IsBlocked(entities []Entity) bool {
	for _, ent := range entities {
		if e.policy.ActionFor(ent.Type) == policy.ActionBlock {
			return true
		}
	}
	return false
}

// VaultRotate re-seals every stored value under the KMS's current key version.
// Call it after promoting a new KEK via the KMS.
func (e *Engine) VaultRotate(ctx context.Context) error {
	return e.vault.Rotate(ctx)
}

// DeAnonymize restores reversible tokens in text back to their original values
// using the session's vault. Non-reversible transforms (FPE/hash/redact) are
// left intact because their originals are not recoverable.
func (e *Engine) DeAnonymize(ctx context.Context, session, text string) string {
	return tokenRe.ReplaceAllStringFunc(text, func(tok string) string {
		v, ok, _ := e.vault.Get(ctx, session, tok)
		if ok {
			return v
		}
		return tok
	})
}

// deterministicHash returns a stable keyed hash of value (not format preserving).
func deterministicHash(value string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(value))
	return "h_" + hex.EncodeToString(mac.Sum(nil))[:16]
}

// FPE (Format-Preserving Encryption, vault-less) deterministically
// transforms value into a stable, format-preserving surrogate. Characters
// keep their class (digit->digit, letter->letter, symbol->symbol) and length,
// so downstream schemas/validations are not broken, but the output is
// irreversible without the key and identical for identical inputs. This is the
// GDPR-friendly "true anonymization" path: no central PII store exists.
func FPE(value, category string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(category))
	mac.Write([]byte{0})
	mac.Write([]byte(value))
	sum := mac.Sum(nil)

	var b strings.Builder
	b.Grow(len(value))
	for i, r := range value {
		switch {
		case r >= '0' && r <= '9':
			d := int(r - '0')
			shift := int(sum[i%len(sum)] % 10)
			b.WriteByte(byte('0' + (d+shift)%10))
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			base := byte('a')
			if r >= 'A' && r <= 'Z' {
				base = 'A'
			}
			shift := int(sum[(i+7)%len(sum)]%26)
			b.WriteByte(base + byte((int(r)-int(base)+shift)%26))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
