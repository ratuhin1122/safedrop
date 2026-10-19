package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type bucket struct {
	limiter *rate.Limiter
	seen    time.Time
}
type Buckets struct {
	mu      sync.Mutex
	entries map[string]*bucket
	limit   rate.Limit
	burst   int
	idle    time.Duration
	sweep   time.Time
	now     func() time.Time
}

func NewBuckets(perSecond float64, burst int) *Buckets {
	if perSecond <= 0 || burst < 1 {
		panic("invalid token bucket configuration")
	}
	idle := time.Duration(float64(time.Second) * float64(burst) / perSecond)
	if idle < time.Minute {
		idle = time.Minute
	}
	return &Buckets{entries: make(map[string]*bucket), limit: rate.Limit(perSecond), burst: burst, idle: idle, now: time.Now}
}

func (b *Buckets) Allow(key string) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	if !now.Before(b.sweep) || len(b.entries) >= 100000 {
		for key, entry := range b.entries {
			if now.Sub(entry.seen) >= b.idle {
				delete(b.entries, key)
			}
		}
		b.sweep = now.Add(b.idle)
	}
	entry := b.entries[key]
	if entry == nil {
		if len(b.entries) >= 100000 {
			return b.idle
		}
		entry = &bucket{limiter: rate.NewLimiter(b.limit, b.burst)}
		b.entries[key] = entry
	}
	entry.seen = now
	if entry.limiter.AllowN(now, 1) {
		return 0
	}
	return time.Duration((1-entry.limiter.TokensAt(now))/float64(b.limit)*float64(time.Second)) + time.Nanosecond
}
