package vault

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"
)

// Vault stores the mapping between an anonymization token and its original
// value, scoped per session. In reversible mode this is the only place the
// real PII lives; it must be encrypted (envelope encryption + KMS) and
// short-lived in production. The MVP ships an in-memory vault and a Redis
// vault; both implement this interface.
type Vault interface {
	// Put stores value under (session, token). ttl bounds how long the
	// mapping survives (re-identification window).
	Put(ctx context.Context, session, token, value string, ttl time.Duration) error
	// Get returns the original value for a token, if present and unexpired.
	Get(ctx context.Context, session, token string) (string, bool, error)
	// Rotate re-seals every stored value under the KMS's current key version.
	// Call it after promoting a new KEK via KMS rotation.
	Rotate(ctx context.Context) error
	// Close releases backend resources.
	Close() error
}

// TLSFiles points at the PEM files needed for mutual TLS between the proxy and
// Redis: the client certificate/key and the CA that issued the server cert.
type TLSFiles struct {
	CA   string // PEM CA used to verify the Redis server certificate
	Cert string // client certificate (for mTLS authentication)
	Key  string // client private key
}

// BuildTLS constructs a *tls.Config for mutual TLS. The client presents its
// certificate (Cert/Key) and verifies the server against CA. Server
// certificate verification is always required (InsecureSkipVerify is false).
func BuildTLS(files TLSFiles) (*tls.Config, error) {
	if files.CA == "" && files.Cert == "" && files.Key == "" {
		return nil, nil
	}
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if files.CA != "" {
		pem, err := os.ReadFile(files.CA)
		if err != nil {
			return nil, fmt.Errorf("vault: read CA: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("vault: invalid CA PEM in %s", files.CA)
		}
		cfg.RootCAs = pool
	}
	if files.Cert != "" || files.Key != "" {
		if files.Cert == "" || files.Key == "" {
			return nil, fmt.Errorf("vault: both client cert and key are required for mTLS")
		}
		cert, err := tls.LoadX509KeyPair(files.Cert, files.Key)
		if err != nil {
			return nil, fmt.Errorf("vault: load client cert: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}
