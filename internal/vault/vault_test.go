package vault

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRotatingKMSEncryptsAndRotates(t *testing.T) {
	master := []byte("master-key-32-bytes-long-abcdefghij")
	kms, err := NewRotatingKMS(master, 1)
	if err != nil {
		t.Fatal(err)
	}
	pt := []byte("secret-PII-value-99887766")
	env, err := Seal(context.Background(), kms, pt)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := MarshalEnvelope(env)
	if strings.Contains(string(b), "secret-PII") {
		t.Fatal("envelope leaks plaintext")
	}
	got, err := Open(context.Background(), kms, env)
	if err != nil || string(got) != string(pt) {
		t.Fatalf("open v1 failed: %v %q", err, got)
	}
	// Rotate to v2; the v1 envelope must still decrypt (old key retained).
	if v := kms.Rotate(); v != 2 {
		t.Fatalf("expected version 2, got %d", v)
	}
	got, err = Open(context.Background(), kms, env)
	if err != nil || string(got) != string(pt) {
		t.Fatalf("v1 envelope must remain readable after rotation: %v", err)
	}
	// New seals use the current (v2) key and decrypt fine.
	env2, _ := Seal(context.Background(), kms, pt)
	got2, err := Open(context.Background(), kms, env2)
	if err != nil || string(got2) != string(pt) {
		t.Fatal("v2 seal/open mismatch")
	}
}

func TestMemoryVaultRotate(t *testing.T) {
	master := []byte("master-key-32-bytes-long-abcdefghij")
	kms, _ := NewRotatingKMS(master, 1)
	v := NewMemoryWithKMS(time.Minute, kms)
	defer v.Close()
	if err := v.Put(context.Background(), "s1", "t1", "pii-value", time.Minute); err != nil {
		t.Fatal(err)
	}
	val, ok, _ := v.Get(context.Background(), "s1", "t1")
	if !ok || val != "pii-value" {
		t.Fatalf("get before rotate failed: %q", val)
	}
	// Promote a new KEK and re-seal the vault.
	kms.Rotate()
	if err := v.Rotate(context.Background()); err != nil {
		t.Fatal(err)
	}
	val, ok, _ = v.Get(context.Background(), "s1", "t1")
	if !ok || val != "pii-value" {
		t.Fatalf("get after rotate failed (re-seal broken): %q", val)
	}
}

// --- mTLS config builder test -------------------------------------------

func genCA(t *testing.T) (certPEM, keyPEM []byte, ca *x509.Certificate, caKey *ecdsa.PrivateKey) {
	caKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "aaas-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := x509.ParseCertificate(der)
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, _ := x509.MarshalECPrivateKey(caKey)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, c, caKey
}

func genLeaf(t *testing.T, ca *x509.Certificate, caKey *ecdsa.PrivateKey, client bool) (certPEM, keyPEM []byte) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "aaas-test-leaf"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if !client {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, _ := x509.MarshalECPrivateKey(key)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}

func TestBuildTLS(t *testing.T) {
	caCert, _, ca, caKey := genCA(t)
	clientCert, clientKey := genLeaf(t, ca, caKey, true)

	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	certPath := filepath.Join(dir, "client.pem")
	keyPath := filepath.Join(dir, "client.key")
	_ = os.WriteFile(caPath, caCert, 0o600)
	_ = os.WriteFile(certPath, clientCert, 0o600)
	_ = os.WriteFile(keyPath, clientKey, 0o600)

	cfg, err := BuildTLS(TLSFiles{CA: caPath, Cert: certPath, Key: keyPath})
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil tls.Config")
	}
	if len(cfg.Certificates) != 1 {
		t.Fatalf("client certificate not loaded: %d", len(cfg.Certificates))
	}
	if cfg.RootCAs == nil {
		t.Fatal("CA pool not set")
	}
	if cfg.InsecureSkipVerify {
		t.Fatal("server cert verification must stay enabled for mTLS")
	}

	// No certs => nil config (plain TCP).
	none, err := BuildTLS(TLSFiles{})
	if err != nil || none != nil {
		t.Fatalf("expected nil tls.Config when no files given: %v %v", none, err)
	}
}
