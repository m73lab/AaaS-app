package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"
)

// KMS wraps/unwraps a data key. In production this fronts an HSM or a managed
// service (AWS KMS, HashiCorp Vault Transit, GCP KMS); the LocalKMS below is a
// stand-in that uses a master key held in process memory (load it from an HSM
// or secret manager, never hardcode it).
type KMS interface {
	Wrap(ctx context.Context, plaintext []byte) ([]byte, error)
	Unwrap(ctx context.Context, wrapped []byte) ([]byte, error)
}

// Rotator is implemented by KMS backends that support key rotation. Rotate
// promotes a new current key version; previously wrapped material remains
// decryptable because each envelope records the version that wrapped it.
type Rotator interface {
	Rotate() int
	CurrentVersion() int
}

// LocalKMS is an AES-GCM key-encrypting-key based KMS. Suitable for local/dev
// and as a reference implementation; for production swap in a cloud/HSM KMS.
type LocalKMS struct {
	kek []byte
}

// NewLocalKMS builds a KMS from a 16/24/32-byte master key.
func NewLocalKMS(kek []byte) (*LocalKMS, error) {
	if l := len(kek); l != 16 && l != 24 && l != 32 {
		return nil, fmt.Errorf("vault: master key must be 16/24/32 bytes, got %d", l)
	}
	return &LocalKMS{kek: kek}, nil
}

func newGCM(kek []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (k *LocalKMS) Wrap(ctx context.Context, plaintext []byte) ([]byte, error) {
	aead, err := newGCM(k.kek)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (k *LocalKMS) Unwrap(ctx context.Context, wrapped []byte) ([]byte, error) {
	aead, err := newGCM(k.kek)
	if err != nil {
		return nil, err
	}
	ns := aead.NonceSize()
	if len(wrapped) < ns {
		return nil, fmt.Errorf("vault: wrapped key too short")
	}
	nonce, ct := wrapped[:ns], wrapped[ns:]
	return aead.Open(nil, nonce, ct, nil)
}

// RotatingKMS supports key rotation. Each master key version derives a KEK via
// HKDF-style hashing; wrapped blobs are prefixed with their 32-bit version so
// Unwrap always selects the correct KEK. Old versions are retained for
// decryption, enabling zero-downtime rotation and later re-sealing.
type RotatingKMS struct {
	mu      sync.Mutex
	master  []byte
	current int
	keys    map[int][]byte
}

// NewRotatingKMS builds a rotating KMS. currentVersion is the active key
// version (>=1); all versions up to it are retained.
func NewRotatingKMS(master []byte, currentVersion int) (*RotatingKMS, error) {
	if len(master) == 0 {
		return nil, fmt.Errorf("vault: rotating KMS requires a master key")
	}
	if currentVersion < 1 {
		currentVersion = 1
	}
	k := &RotatingKMS{master: master, current: currentVersion, keys: map[int][]byte{}}
	for v := 1; v <= currentVersion; v++ {
		k.keys[v] = kekFromMaster(master, v)
	}
	return k, nil
}

func kekFromMaster(master []byte, version int) []byte {
	h := sha256.New()
	h.Write(master)
	h.Write([]byte("aaas-kek"))
	h.Write([]byte(strconv.Itoa(version)))
	return h.Sum(nil)
}

func (k *RotatingKMS) Wrap(ctx context.Context, plaintext []byte) ([]byte, error) {
	k.mu.Lock()
	kek, v := k.keys[k.current], k.current
	k.mu.Unlock()
	aead, err := newGCM(kek)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	blob := aead.Seal(nonce, nonce, plaintext, nil)
	out := make([]byte, 4+len(blob))
	binary.BigEndian.PutUint32(out[:4], uint32(v))
	copy(out[4:], blob)
	return out, nil
}

func (k *RotatingKMS) Unwrap(ctx context.Context, wrapped []byte) ([]byte, error) {
	if len(wrapped) < 4 {
		return nil, fmt.Errorf("vault: wrapped key missing version header")
	}
	v := int(binary.BigEndian.Uint32(wrapped[:4]))
	k.mu.Lock()
	kek, ok := k.keys[v]
	k.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("vault: KEK version %d unavailable", v)
	}
	aead, err := newGCM(kek)
	if err != nil {
		return nil, err
	}
	blob := wrapped[4:]
	ns := aead.NonceSize()
	if len(blob) < ns {
		return nil, fmt.Errorf("vault: wrapped key too short")
	}
	return aead.Open(nil, blob[:ns], blob[ns:], nil)
}

// Rotate promotes a new current key version and returns it. Old versions stay
// available for decryption.
func (k *RotatingKMS) Rotate() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.current++
	k.keys[k.current] = kekFromMaster(k.master, k.current)
	return k.current
}

// CurrentVersion returns the active key version.
func (k *RotatingKMS) CurrentVersion() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.current
}

// Envelope is the encrypted form stored in the backend. The data is sealed
// under a per-value data key (DEK); the DEK is wrapped by the KMS. This is
// envelope encryption: compromising the backend yields only ciphertext + a
// KEK-wrapped DEK, useless without the KMS.
type Envelope struct {
	WrappedDEK []byte `json:"w"`
	Nonce      []byte `json:"n"`
	Ciphertext []byte `json:"c"`
}

// Seal encrypts plaintext with a fresh random DEK and wraps the DEK via KMS.
func Seal(ctx context.Context, kms KMS, plaintext []byte) (*Envelope, error) {
	dek := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ct := aead.Seal(nil, nonce, plaintext, nil)
	wdek, err := kms.Wrap(ctx, dek)
	if err != nil {
		return nil, err
	}
	return &Envelope{WrappedDEK: wdek, Nonce: nonce, Ciphertext: ct}, nil
}

// Open reverses Seal, recovering the plaintext via the KMS-unwrapped DEK.
func Open(ctx context.Context, kms KMS, env *Envelope) ([]byte, error) {
	dek, err := kms.Unwrap(ctx, env.WrappedDEK)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return aead.Open(nil, env.Nonce, env.Ciphertext, nil)
}

// MarshalEnvelope serializes an envelope to bytes for storage.
func MarshalEnvelope(env *Envelope) ([]byte, error) { return json.Marshal(env) }

// UnmarshalEnvelope parses a stored envelope.
func UnmarshalEnvelope(b []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, err
	}
	return &env, nil
}
