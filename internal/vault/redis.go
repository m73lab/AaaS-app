package vault

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisVault stores token->value mappings in Redis, one key per
// (session, token) with a TTL. When a KMS is configured, values are stored
// envelope-encrypted (see crypto.go) so a Redis compromise yields no PII.
// Enable TLS (and an SSH tunnel to homelab02) so the backend is encrypted in
// transit as well as at rest.
type RedisVault struct {
	client *redis.Client
	prefix string
	kms    KMS
}

// RedisOptions configures the Redis vault.
type RedisOptions struct {
	Addr     string
	Password string
	DB       int
	TLS      *tls.Config
	KMS      KMS
}

// NewRedis creates a Redis-backed vault. Enable TLS in transit and supply a
// KMS for envelope encryption in production.
func NewRedis(opts RedisOptions) *RedisVault {
	return &RedisVault{
		client: redis.NewClient(&redis.Options{
			Addr:     opts.Addr,
			Password: opts.Password,
			DB:       opts.DB,
			TLSConfig: opts.TLS,
		}),
		prefix: "aaas",
		kms:    opts.KMS,
	}
}

func (r *RedisVault) key(session, token string) string {
	return fmt.Sprintf("%s:%s:%s", r.prefix, session, token)
}

func (r *RedisVault) Put(ctx context.Context, session, token, value string, ttl time.Duration) error {
	stored := []byte(value)
	if r.kms != nil {
		env, err := Seal(ctx, r.kms, []byte(value))
		if err != nil {
			return err
		}
		stored, err = MarshalEnvelope(env)
		if err != nil {
			return err
		}
	}
	return r.client.Set(ctx, r.key(session, token), stored, ttl).Err()
}

func (r *RedisVault) Get(ctx context.Context, session, token string) (string, bool, error) {
	raw, err := r.client.Get(ctx, r.key(session, token)).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if r.kms == nil {
		return raw, true, nil
	}
	env, err := UnmarshalEnvelope([]byte(raw))
	if err != nil {
		return "", false, err
	}
	pt, err := Open(ctx, r.kms, env)
	if err != nil {
		return "", false, err
	}
	return string(pt), true, nil
}

func (r *RedisVault) Close() error {
	return r.client.Close()
}

// Rotate re-seals every stored value under the KMS's current key version,
// preserving each key's TTL. It is a no-op when no KMS is configured.
func (r *RedisVault) Rotate(ctx context.Context) error {
	if r.kms == nil {
		return nil
	}
	var cursor uint64
	for {
		keys, next, err := r.client.Scan(ctx, cursor, r.prefix+":*", 100).Result()
		if err != nil {
			return err
		}
		for _, key := range keys {
			ttl, err := r.client.TTL(ctx, key).Result()
			if err != nil {
				return err
			}
			raw, err := r.client.Get(ctx, key).Result()
			if err == redis.Nil {
				continue
			}
			if err != nil {
				return err
			}
			env, err := UnmarshalEnvelope([]byte(raw))
			if err != nil {
				return err
			}
			pt, err := Open(ctx, r.kms, env)
			if err != nil {
				return err
			}
			newEnv, err := Seal(ctx, r.kms, pt)
			if err != nil {
				return err
			}
			b, err := MarshalEnvelope(newEnv)
			if err != nil {
				return err
			}
			if ttl < 0 {
				ttl = 0
			}
			if err := r.client.Set(ctx, key, b, ttl).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}
