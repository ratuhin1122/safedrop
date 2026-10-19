package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrChallenge = errors.New("registration verification failed")
var ErrChallengeUnavailable = errors.New("registration verification unavailable")

type RegistrationCheck interface {
	Verify(context.Context, string, string) error
}
type Turnstile struct {
	Secret, Hostname string
	client           *http.Client
	endpoint         string
}

func NewTurnstile(secret, hostname string) *Turnstile {
	return &Turnstile{Secret: secret, Hostname: hostname,
		client:   &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		endpoint: "https://challenges.cloudflare.com/turnstile/v0/siteverify"}
}

func (t *Turnstile) Verify(ctx context.Context, token, ip string) error {
	if token == "" || len(token) > 2048 {
		return ErrChallenge
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	form := url.Values{"secret": {t.Secret}, "response": {token}, "remoteip": {ip}}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return ErrChallengeUnavailable
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := t.client.Do(r)
	if err != nil {
		return ErrChallengeUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ErrChallengeUnavailable
	}
	var result struct {
		Success  bool   `json:"success"`
		Hostname string `json:"hostname"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 16385))
	if decoder.Decode(&result) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return ErrChallengeUnavailable
	}
	if !result.Success || (t.Hostname != "" && result.Hostname != t.Hostname) {
		return ErrChallenge
	}
	return nil
}
