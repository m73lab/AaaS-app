package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"aaas/internal/policy"
)

// Config is the top-level AaaS proxy configuration.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Upstream  UpstreamConfig  `yaml:"upstream"`
	Vault     VaultConfig     `yaml:"vault"`
	Policy    PolicyConfig    `yaml:"policy"`
	Detect    DetectConfig    `yaml:"detect"`
	Audit     AuditConfig     `yaml:"audit"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Dashboard DashboardConfig `yaml:"dashboard"`
}

// LimitPair holds per-window request ceilings.
type LimitPair struct {
	PerMinute int `yaml:"per_minute"`
	PerHour   int `yaml:"per_hour"`
}

// RateLimitConfig controls per-tenant request ceilings.
type RateLimitConfig struct {
	Enabled bool                `yaml:"enabled"`
	Default LimitPair           `yaml:"default"`
	Tenants map[string]LimitPair `yaml:"tenants"` // tenant id -> overrides
}

// DashboardConfig points the proxy at the dashboard backend used to record
// per-request usage analytics.
type DashboardConfig struct {
	UsageURL string `yaml:"usage_url"` // e.g. http://dashboard:3001/v1/aas/usage-logs
	Enabled  bool   `yaml:"enabled"`
}

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type UpstreamConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Timeout int    `yaml:"timeout"`
}

type VaultConfig struct {
	Type           string   `yaml:"type"` // memory | redis
	RedisAddr      string   `yaml:"redis_addr"`
	RedisPassword  string   `yaml:"redis_password"`
	RedisDB        int      `yaml:"redis_db"`
	RedisTLS       bool     `yaml:"redis_tls"`
	RedisClientCert string  `yaml:"redis_client_cert"` // mTLS client cert (PEM)
	RedisClientKey  string  `yaml:"redis_client_key"`  // mTLS client key (PEM)
	RedisCA         string  `yaml:"redis_ca_cert"`     // CA to verify server (PEM)
	TTLSeconds     int      `yaml:"ttl_seconds"`
	KMS            KMSConfig `yaml:"kms"`
}

// KMSConfig selects the key-management backend for envelope encryption.
// Type "local" uses a master key from MasterKeyEnv (a stand-in for an HSM /
// cloud KMS). Type "rotating" derives versioned KEKs from the master key and
// supports rotation (old versions retained for decryption). Additional types
// (aws-kms, vault-transit) plug in via the KMS interface in internal/vault.
type KMSConfig struct {
	Type          string `yaml:"type"` // local | rotating | none
	MasterKeyEnv  string `yaml:"master_key_env"`
	VersionEnv    string `yaml:"version_env"` // env var with current KEK version (rotating)
}

type PolicyConfig struct {
	DefaultAction string       `yaml:"default_action"`
	Rules         []RuleConfig `yaml:"rules"`
	// FailClosed rejects the request when detection fails (instead of
	// forwarding un-anonymized text).
	FailClosed    bool `yaml:"fail_closed"`
}

type RuleConfig struct {
	Categories []string `yaml:"categories"`
	Action     string   `yaml:"action"`
}

type DetectConfig struct {
	Dictionary         []string `yaml:"dictionary"`
	EnablePhone        bool     `yaml:"enable_phone"`
	PersonNames        []string `yaml:"person_names"`
	HeuristicNER       bool     `yaml:"heuristic_ner"`
	PresidioURL        string   `yaml:"presidio_url"`
	PresidioLang       string   `yaml:"presidio_lang"`
	PresidioCA         string   `yaml:"presidio_ca"`           // CA to verify Presidio server (mTLS)
	PresidioClientCert string   `yaml:"presidio_client_cert"`  // client cert for mTLS to Presidio
	PresidioClientKey  string   `yaml:"presidio_client_key"`
}

type AuditConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"` // empty => stderr
}

// Load reads and parses the YAML config, applying safe defaults.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	c.setDefaults()
	return &c, nil
}

func (c *Config) setDefaults() {
	if c.Server.Listen == "" {
		c.Server.Listen = ":8080"
	}
	if c.Upstream.Timeout == 0 {
		c.Upstream.Timeout = 30
	}
	if c.Vault.Type == "" {
		c.Vault.Type = "memory"
	}
	if c.Vault.TTLSeconds == 0 {
		c.Vault.TTLSeconds = 300
	}
	if c.Policy.DefaultAction == "" {
		c.Policy.DefaultAction = string(policy.ActionReversible)
	}
	if c.RateLimit.Enabled && c.RateLimit.Default.PerMinute == 0 && c.RateLimit.Default.PerHour == 0 {
		c.RateLimit.Default = LimitPair{PerMinute: 60, PerHour: 2000}
	}
	if c.Dashboard.UsageURL != "" {
		c.Dashboard.Enabled = true
	}
}

// RuntimePolicy builds the runtime policy from the configuration.
func (c *Config) RuntimePolicy() *policy.Policy {
	p := &policy.Policy{Default: policy.Action(c.Policy.DefaultAction)}
	for _, r := range c.Policy.Rules {
		p.Rules = append(p.Rules, policy.Rule{
			Categories: r.Categories,
			Action:     policy.Action(r.Action),
		})
	}
	return p
}

// TTL returns the vault TTL as a duration.
func (c *Config) TTL() time.Duration {
	return time.Duration(c.Vault.TTLSeconds) * time.Second
}
