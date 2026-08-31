package main

import (
	"crypto/sha256"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"aaas/internal/audit"
	"aaas/internal/config"
	"aaas/internal/detect"
	"aaas/internal/proxy"
	"aaas/internal/transform"
	"aaas/internal/vault"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	// --- Master key + KMS (envelope encryption) -------------------------
	master := masterKey(cfg)
	var kms vault.KMS
	switch cfg.Vault.KMS.Type {
	case "local":
		if master == nil {
			log.Printf("WARNING: KMS type 'local' but no master key found; values stored unencrypted")
			break
		}
		kms, err = vault.NewLocalKMS(master)
		if err != nil {
			log.Fatalf("kms: %v", err)
		}
		log.Printf("KMS: local envelope encryption enabled (master from %s)", cfg.Vault.KMS.MasterKeyEnv)
	case "rotating":
		if master == nil {
			log.Fatalf("kms: rotating KMS requires a master key (%s)", cfg.Vault.KMS.MasterKeyEnv)
		}
		ver := keyVersion(cfg.Vault.KMS.VersionEnv)
		kms, err = vault.NewRotatingKMS(master, ver)
		if err != nil {
			log.Fatalf("kms: %v", err)
		}
		log.Printf("KMS: rotating envelope encryption enabled (current version %d)", kms.(vault.Rotator).CurrentVersion())
	default:
		if cfg.Vault.Type == "redis" {
			log.Printf("WARNING: Redis vault without a KMS stores PII in plaintext — configure vault.kms.type=local|rotating")
		}
	}

	// FPE key derived from the master key (vault-less pseudonymization).
	fpeKey := deriveKey(master)

	// --- Detection ensemble --------------------------------------------
	detectors := []detect.Detector{}
	detectors = append(detectors, detect.StructuredDetectors(cfg.Detect.EnablePhone)...)
	if len(cfg.Detect.PersonNames) > 0 {
		detectors = append(detectors, detect.NewDictionaryDetector("PERSON", cfg.Detect.PersonNames, 0.8))
	}
	if len(cfg.Detect.Dictionary) > 0 {
		detectors = append(detectors, detect.NewDictionaryDetector("CUSTOM", cfg.Detect.Dictionary, 0.85))
	}
	// Real NER: Microsoft Presidio (ONNX-backed) sidecar when configured,
	// with mutual TLS when certs are supplied.
	detectors = append(detectors, detect.NewPresidioNER(
		cfg.Detect.PresidioURL, cfg.Detect.PresidioLang,
		cfg.Detect.PresidioCA, cfg.Detect.PresidioClientCert, cfg.Detect.PresidioClientKey,
	))
	// Interim dependency-free NER to lift recall before Presidio is wired.
	detectors = append(detectors, detect.NewHeuristicNER(cfg.Detect.HeuristicNER, "PERSON", 0.5))
	combined := detect.NewCombined(detectors...)

	// --- Vault ----------------------------------------------------------
	var v vault.Vault
	switch cfg.Vault.Type {
	case "redis":
		tlsCfg, terr := vault.BuildTLS(vault.TLSFiles{
			CA:   cfg.Vault.RedisCA,
			Cert: cfg.Vault.RedisClientCert,
			Key:  cfg.Vault.RedisClientKey,
		})
		if terr != nil {
			log.Fatalf("vault tls: %v", terr)
		}
		if cfg.Vault.RedisTLS && tlsCfg == nil {
			log.Printf("WARNING: redis_tls=true but no CA/cert provided; using plain TCP")
		}
		log.Printf("vault: redis at %s (tls=%v, envelope=%v)", cfg.Vault.RedisAddr, tlsCfg != nil, kms != nil)
		v = vault.NewRedis(vault.RedisOptions{
			Addr:     cfg.Vault.RedisAddr,
			Password: cfg.Vault.RedisPassword,
			DB:       cfg.Vault.RedisDB,
			TLS:      tlsCfg,
			KMS:      kms,
		})
	default:
		if kms != nil {
			v = vault.NewMemoryWithKMS(30*time.Second, kms)
		} else {
			v = vault.NewMemory(30 * time.Second)
		}
	}
	defer v.Close()

	engine := transform.New(combined, v, cfg.RuntimePolicy(), fpeKey, cfg.TTL())

	// --- Audit ----------------------------------------------------------
	var auditor *audit.Auditor
	if cfg.Audit.Enabled {
		auditor, err = audit.New(cfg.Audit.Path)
		if err != nil {
			log.Fatalf("audit: %v", err)
		}
		defer auditor.Close()
		log.Printf("audit: enabled (path=%q)", cfg.Audit.Path)
	}

	upstream := proxy.NewUpstream(cfg.Upstream.BaseURL, cfg.Upstream.APIKey, time.Duration(cfg.Upstream.Timeout)*time.Second)
	srv := proxy.NewServer(cfg.Server.Listen, engine, upstream, auditor, cfg.Policy.FailClosed, kms, cfg.RateLimit, cfg.Dashboard)
	log.Fatal(srv.Start())
}

// masterKey resolves the envelope-encryption master key from the configured
// environment variable (or AaaS_MASTER_KEY), normalizing to 32 bytes. In
// production this should come from an HSM / secret manager, never a literal.
func masterKey(cfg *config.Config) []byte {
	env := cfg.Vault.KMS.MasterKeyEnv
	if env == "" {
		env = "AaaS_MASTER_KEY"
	}
	raw := os.Getenv(env)
	if raw == "" {
		raw = os.Getenv("AaaS_MASTER_KEY")
	}
	if raw == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

// deriveKey turns an arbitrary master key (or dev fallback) into a 32-byte key.
func deriveKey(master []byte) []byte {
	seed := master
	if seed == nil {
		seed = []byte("dev-insecure-fpe-key-change-me")
	}
	sum := sha256.Sum256(append(seed, []byte(":fpe")...))
	return sum[:]
}

// keyVersion resolves the current KEK version from the configured env var
// (default AaaS_KEK_VERSION), used by the rotating KMS.
func keyVersion(env string) int {
	if env == "" {
		env = "AaaS_KEK_VERSION"
	}
	raw := os.Getenv(env)
	if raw == "" {
		return 1
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return 1
	}
	return v
}
