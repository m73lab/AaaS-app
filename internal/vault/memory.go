package vault

import (
	"context"
	"sync"
	"time"
)

// Memory is an in-process vault keyed by session then token, with lazy TTL
// expiration. When a KMS is supplied, values are stored envelope-encrypted
// (see crypto.go). Suitable for single-instance MVP, local testing and unit
// tests of rotation; for multi-instance/persistent deployments use RedisVault.
type Memory struct {
	mu      sync.RWMutex
	data    map[string]map[string]entry
	kms     KMS
	now     func() time.Time
	stop    chan struct{}
	cleanup time.Duration
}

type entry struct {
	value  string
	expire time.Time
}

// NewMemory creates an in-memory vault (plaintext storage) that evicts expired
// entries every cleanup interval.
func NewMemory(cleanup time.Duration) *Memory {
	return newMemory(cleanup, nil)
}

// NewMemoryWithKMS creates an in-memory vault that envelope-encrypts values
// with the supplied KMS (used to exercise rotation without Redis).
func NewMemoryWithKMS(cleanup time.Duration, kms KMS) *Memory {
	return newMemory(cleanup, kms)
}

func newMemory(cleanup time.Duration, kms KMS) *Memory {
	if cleanup <= 0 {
		cleanup = 30 * time.Second
	}
	m := &Memory{
		data:    make(map[string]map[string]entry),
		kms:     kms,
		now:     time.Now,
		stop:    make(chan struct{}),
		cleanup: cleanup,
	}
	go m.sweep()
	return m
}

func (m *Memory) sweep() {
	ticker := time.NewTicker(m.cleanup)
	defer ticker.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			now := m.now()
			m.mu.Lock()
			for sid, tokens := range m.data {
				for tok, e := range tokens {
					if now.After(e.expire) {
						delete(tokens, tok)
					}
				}
				if len(tokens) == 0 {
					delete(m.data, sid)
				}
			}
			m.mu.Unlock()
		}
	}
}

func (m *Memory) Put(ctx context.Context, session, token, value string, ttl time.Duration) error {
	stored := value
	if m.kms != nil {
		env, err := Seal(ctx, m.kms, []byte(value))
		if err != nil {
			return err
		}
		b, err := MarshalEnvelope(env)
		if err != nil {
			return err
		}
		stored = string(b)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data[session] == nil {
		m.data[session] = make(map[string]entry)
	}
	m.data[session][token] = entry{value: stored, expire: m.now().Add(ttl)}
	return nil
}

func (m *Memory) Get(ctx context.Context, session, token string) (string, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tokens, ok := m.data[session]
	if !ok {
		return "", false, nil
	}
	e, ok := tokens[token]
	if !ok || m.now().After(e.expire) {
		return "", false, nil
	}
	if m.kms == nil {
		return e.value, true, nil
	}
	env, err := UnmarshalEnvelope([]byte(e.value))
	if err != nil {
		return "", false, err
	}
	pt, err := Open(ctx, m.kms, env)
	if err != nil {
		return "", false, err
	}
	return string(pt), true, nil
}

// Rotate re-seals every stored value under the KMS's current key version.
func (m *Memory) Rotate(ctx context.Context) error {
	if m.kms == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for sid, tokens := range m.data {
		for tok, e := range tokens {
			env, err := UnmarshalEnvelope([]byte(e.value))
			if err != nil {
				return err
			}
			pt, err := Open(ctx, m.kms, env)
			if err != nil {
				return err
			}
			newEnv, err := Seal(ctx, m.kms, pt)
			if err != nil {
				return err
			}
			b, err := MarshalEnvelope(newEnv)
			if err != nil {
				return err
			}
			tokens[tok] = entry{value: string(b), expire: e.expire}
		}
		_ = sid
	}
	return nil
}

func (m *Memory) Close() error {
	close(m.stop)
	return nil
}
