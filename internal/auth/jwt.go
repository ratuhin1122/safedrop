package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const tokenHeader = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
const tokenIssuer = "safedrop"

type Tokens struct{ key []byte }
type claims struct {
	Subject  string `json:"sub"`
	Issuer   string `json:"iss"`
	Audience string `json:"aud"`
	Issued   int64  `json:"iat"`
	Expires  int64  `json:"exp"`
}

func NewTokens(secret string) (*Tokens, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must contain at least 32 bytes")
	}
	return &Tokens{key: []byte(secret)}, nil
}

func (t *Tokens) sign(message string) []byte {
	m := hmac.New(sha256.New, t.key)
	_, _ = m.Write([]byte(message))
	return m.Sum(nil)
}

func (t *Tokens) Issue(id pgtype.UUID) string {
	now := time.Now().Unix()
	payload, _ := json.Marshal(claims{Subject: id.String(), Issuer: tokenIssuer,
		Audience: "safedrop-api", Issued: now, Expires: now + 900})
	message := tokenHeader + "." + base64.RawURLEncoding.EncodeToString(payload)
	return message + "." + base64.RawURLEncoding.EncodeToString(t.sign(message))
}

func (t *Tokens) Authenticate(r *http.Request) (pgtype.UUID, error) {
	var id pgtype.UUID
	values := r.Header.Values("Authorization")
	if len(values) != 1 || len(values[0]) > 2048 {
		return id, ErrUnauthorized
	}
	fields := strings.Fields(values[0])
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return id, ErrUnauthorized
	}
	parts := strings.Split(fields[1], ".")
	if len(parts) != 3 || parts[0] != tokenHeader {
		return id, ErrUnauthorized
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(sig, t.sign(parts[0]+"."+parts[1])) {
		return id, ErrUnauthorized
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return id, ErrUnauthorized
	}
	var c claims
	now := time.Now().Unix()
	if json.Unmarshal(payload, &c) != nil || c.Issuer != tokenIssuer || c.Audience != "safedrop-api" ||
		c.Expires <= now || c.Issued > now || c.Expires-c.Issued != 900 {
		return id, ErrUnauthorized
	}
	if id.Scan(c.Subject) != nil || !id.Valid || id.Bytes == [16]byte{} {
		return pgtype.UUID{}, ErrUnauthorized
	}
	return id, nil
}
