package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSlidingWindow(t *testing.T) {
	for _, dimension := range []string{"user", "ip"} {
		t.Run(dimension, func(t *testing.T) {
			u := NewUploads(UploadConfig{Window: time.Minute, PerUser: 20, PerIP: 20})
			now := time.Unix(1000, 0)
			u.now = func() time.Time { return now }
			for i := 0; i < 20; i++ {
				user, ip := "user", "ip"
				if dimension == "user" {
					ip = fmt.Sprint(i)
				} else {
					user = fmt.Sprint(i)
				}
				release, err := u.Begin(context.Background(), user, ip)
				if err != nil {
					t.Fatalf("upload %d: %v", i+1, err)
				}
				release()
				release()
			}
			_, err := u.Begin(context.Background(), "user", "ip")
			var denied *Rejection
			if !errors.As(err, &denied) || denied.RetryAfter != time.Minute {
				t.Fatalf("21st: %v", err)
			}
			now = now.Add(time.Minute)
			release, err := u.Begin(context.Background(), "user", "ip")
			if err != nil {
				t.Fatal(err)
			}
			release()
		})
	}
}

func TestConcurrency(t *testing.T) {
	u := NewUploads(UploadConfig{Window: time.Minute, PerUser: 100, PerIP: 100})
	var wg sync.WaitGroup
	var mu sync.Mutex
	var releases []func()
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := u.Begin(context.Background(), "user", "ip")
			if err == nil {
				mu.Lock()
				releases = append(releases, release)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if len(releases) != 3 {
		t.Fatalf("active uploads = %d", len(releases))
	}
	for _, release := range releases {
		release()
	}
	release, err := u.Begin(context.Background(), "user", "ip")
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestWindowDoesNotResetAtClockBoundary(t *testing.T) {
	u := NewUploads(UploadConfig{Window: time.Minute, PerUser: 1, PerIP: 1})
	now := time.Unix(59, 0)
	u.now = func() time.Time { return now }
	release, _ := u.Begin(context.Background(), "user", "ip")
	release()
	now = time.Unix(61, 0)
	_, err := u.Begin(context.Background(), "user", "ip")
	var denied *Rejection
	if !errors.As(err, &denied) || denied.RetryAfter != 58*time.Second {
		t.Fatalf("sliding window: %v", err)
	}
}
