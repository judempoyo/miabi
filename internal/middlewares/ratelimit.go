// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/jkaninda/logger"
	"github.com/jkaninda/okapi"
	"github.com/redis/go-redis/v9"
)

// RateLimit returns a fixed-window rate-limit middleware keyed by client IP and
// request path, backed by Redis. It is intended for unauthenticated, abuse-prone
// endpoints (login, registration, password reset).
//
// When Redis is unavailable, behavior depends on localFallback:
//   - localFallback=false — fail OPEN (allow). Right for availability-critical,
//     non-brute-forceable endpoints (e.g. token-authenticated agent tunnels),
//     where locking clients out during a Redis blip is worse than losing throttling.
//   - localFallback=true — fall back to a per-instance IN-MEMORY limiter instead
//     of failing open, so a Redis outage can't silently re-enable brute-force on
//     auth endpoints, yet legitimate users can still sign in. The local limiter is
//     per-process (coarser than the shared Redis window in a multi-instance
//     deploy) but strictly better than no limit — and exact on a single node.
func RateLimit(rdb *redis.Client, limit int, window time.Duration, localFallback bool) okapi.Middleware {
	var (
		fallback *memLimiter
		lastWarn atomic.Int64 // unix nanos of the last "redis down" log, to throttle it
	)
	if localFallback {
		fallback = newMemLimiter()
	}
	warnOnce := func(err error) {
		now := time.Now().UnixNano()
		if prev := lastWarn.Load(); now-prev > int64(30*time.Second) && lastWarn.CompareAndSwap(prev, now) {
			logger.Warn("rate limiter: Redis unavailable, using in-memory fallback", "error", err)
		}
	}

	return func(c *okapi.Context) error {
		if limit <= 0 {
			return c.Next()
		}
		if rdb == nil {
			// No Redis wired at all (not an outage): use the local fallback when one
			// is configured so the endpoint is still throttled; otherwise allow.
			if fallback != nil {
				return applyMemLimit(c, fallback, limit, window)
			}
			return c.Next()
		}
		ctx := c.Request().Context()
		key := "rl:" + c.Request().URL.Path + ":" + c.RealIP()

		n, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			if fallback != nil {
				warnOnce(err)
				return applyMemLimit(c, fallback, limit, window)
			}
			return c.Next() // fail open
		}
		if n == 1 {
			_ = rdb.Expire(ctx, key, window).Err()
		}
		if n > int64(limit) {
			return c.AbortTooManyRequests("too many requests — please slow down and try again shortly")
		}
		return c.Next()
	}
}

func applyMemLimit(c *okapi.Context, m *memLimiter, limit int, window time.Duration) error {
	if !m.allow(c.Request().URL.Path+":"+c.RealIP(), limit, window) {
		return c.AbortTooManyRequests("too many requests — please slow down and try again shortly")
	}
	return c.Next()
}

// memLimiter is a small fixed-window per-process rate limiter used as the
// fallback when Redis is down. Windows expire on their own; the map is swept
// opportunistically once it grows past a bound so it can't leak memory.
type memLimiter struct {
	mu   sync.Mutex
	hits map[string]memWindow
}

type memWindow struct {
	count   int
	resetAt time.Time
}

func newMemLimiter() *memLimiter { return &memLimiter{hits: make(map[string]memWindow)} }

// memLimiterSweepAt is the map size that triggers eviction of expired windows.
const memLimiterSweepAt = 10000

func (m *memLimiter) allow(key string, limit int, window time.Duration) bool {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.hits) > memLimiterSweepAt {
		for k, w := range m.hits {
			if now.After(w.resetAt) {
				delete(m.hits, k)
			}
		}
	}
	w, ok := m.hits[key]
	if !ok || now.After(w.resetAt) {
		m.hits[key] = memWindow{count: 1, resetAt: now.Add(window)}
		return true
	}
	w.count++
	m.hits[key] = w
	return w.count <= limit
}
