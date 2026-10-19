package ratelimit

import (
	"context"
	"sync"
	"time"
)

type UploadConfig struct {
	Window         time.Duration
	PerUser, PerIP int
}

type Uploads struct {
	mu     sync.Mutex
	cfg    UploadConfig
	events map[string][]time.Time
	active map[string]int
	sweep  time.Time
	now    func() time.Time
}

func NewUploads(cfg UploadConfig) *Uploads {
	if cfg.Window <= 0 || cfg.PerUser < 1 || cfg.PerIP < 1 {
		panic("invalid upload limit configuration")
	}
	return &Uploads{cfg: cfg, events: make(map[string][]time.Time), active: make(map[string]int), now: time.Now}
}

func (u *Uploads) prune(key string, now time.Time) []time.Time {
	times := u.events[key]
	i := 0
	for i < len(times) && !times[i].Add(u.cfg.Window).After(now) {
		i++
	}
	if i == len(times) {
		delete(u.events, key)
		return nil
	}
	u.events[key] = times[i:]
	return times[i:]
}

func (u *Uploads) Begin(ctx context.Context, user, ip string) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	now := u.now()
	if !now.Before(u.sweep) || len(u.events) >= 100000 {
		for key := range u.events {
			u.prune(key, now)
		}
		u.sweep = now.Add(u.cfg.Window)
	}
	keys := []string{"user:" + user, "ip:" + ip}
	limits := []int{u.cfg.PerUser, u.cfg.PerIP}
	retry := time.Duration(0)
	for i, key := range keys {
		times := u.prune(key, now)
		if len(times) >= limits[i] {
			if wait := times[len(times)-limits[i]].Add(u.cfg.Window).Sub(now); wait > retry {
				retry = wait
			}
		}
	}
	if retry > 0 {
		return nil, &Rejection{RetryAfter: retry, Kind: "upload_window"}
	}
	if u.active[user] >= 3 {
		return nil, &Rejection{RetryAfter: time.Second, Kind: "concurrency"}
	}
	if len(u.events)+2 > 100000 {
		return nil, &Rejection{RetryAfter: u.cfg.Window, Kind: "upload_window"}
	}
	for _, key := range keys {
		u.events[key] = append(u.events[key], now)
	}
	u.active[user]++
	var once sync.Once
	return func() {
		once.Do(func() {
			u.mu.Lock()
			defer u.mu.Unlock()
			u.active[user]--
			if u.active[user] == 0 {
				delete(u.active, user)
			}
		})
	}, nil
}
