package proxy

import (
	"sync"
	"time"

	"aaas/internal/config"
)

// rateLimiter enforces per-tenant sliding-window ceilings for the minute and
// hour windows. It is intended for a single proxy instance (the homelab
// deployment runs one container); for horizontally scaled deployments swap the
// backing store for Redis while keeping this interface.
type rateLimiter struct {
	mu       sync.Mutex
	buckets  map[string][]time.Time // tenant -> request timestamps (both windows)
	defaults config.LimitPair
	overrides map[string]config.LimitPair
}

func newRateLimiter(cfg config.RateLimitConfig) *rateLimiter {
	ov := cfg.Tenants
	if ov == nil {
		ov = map[string]config.LimitPair{}
	}
	return &rateLimiter{
		buckets:   map[string][]time.Time{},
		defaults:  cfg.Default,
		overrides: ov,
	}
}

func (rl *rateLimiter) limits(tenant string) (perMin, perHour int) {
	if o, ok := rl.overrides[tenant]; ok {
		return o.PerMinute, o.PerHour
	}
	return rl.defaults.PerMinute, rl.defaults.PerHour
}

// allow records a request and reports whether it is within limits.
func (rl *rateLimiter) allow(tenant string, now time.Time) (ok bool, perMin int, perHour int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limitMin, limitHour := rl.limits(tenant)
	cutMin := now.Add(-time.Minute)
	cutHour := now.Add(-time.Hour)

	ts := rl.buckets[tenant]
	kept := ts[:0]
	minCount, hourCount := 0, 0
	for _, t := range ts {
		if t.After(cutHour) {
			kept = append(kept, t)
			hourCount++
			if t.After(cutMin) {
				minCount++
			}
		}
	}
	kept = append(kept, now)
	rl.buckets[tenant] = kept

	within := true
	if limitMin > 0 && minCount+1 > limitMin {
		within = false
	}
	if limitHour > 0 && hourCount+1 > limitHour {
		within = false
	}
	return within, minCount + 1, hourCount + 1
}
