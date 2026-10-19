package ratelimit

import (
	"context"
	"math"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"time"
)

type Rejection struct {
	RetryAfter time.Duration
	Kind       string
}

func (e *Rejection) Error() string { return "rate or quota limit exceeded" }

// UploadLimiter can be replaced by a shared Redis implementation.
type UploadLimiter interface {
	Begin(context.Context, string, string) (release func(), err error)
}

func IP(r *http.Request) string {
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return "unknown"
	}
	return ip.Unmap().WithZone("").String()
}

func Respond(w http.ResponseWriter, retry time.Duration) {
	seconds := int64(math.Ceil(retry.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.FormatInt(seconds, 10))
	w.Header().Set("Cache-Control", "no-store")
	http.Error(w, "rate or quota limit exceeded", http.StatusTooManyRequests)
}

func UntilTomorrow(now time.Time) time.Duration {
	u := now.UTC()
	return time.Date(u.Year(), u.Month(), u.Day()+1, 0, 0, 0, 0, time.UTC).Sub(u)
}
