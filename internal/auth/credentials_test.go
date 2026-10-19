package auth

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestPasswords(t *testing.T) {
	hash := hashPassword("long enough password")
	other := hashPassword("long enough password")
	if hash == other || !strings.HasPrefix(hash, passwordPrefix) {
		t.Fatal("password salt or format")
	}
	for _, tc := range []struct {
		hash, password string
		want           bool
	}{
		{hash, "long enough password", true}, {hash, "incorrect password", false},
		{"$argon2id$invalid", "long enough password", false}, {"plaintext", "plaintext", false},
	} {
		if got := checkPassword(tc.hash, tc.password); got != tc.want {
			t.Errorf("verification = %v", got)
		}
	}
}

func TestJWTValidation(t *testing.T) {
	tokens, err := NewTokens(strings.Repeat("secret", 8))
	if err != nil {
		t.Fatal(err)
	}
	var id pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000001")
	valid := tokens.Issue(id)
	expiredPayload, _ := json.Marshal(claims{Subject: id.String(), Issuer: tokenIssuer, Audience: "safedrop-api",
		Issued: time.Now().Unix() - 1000, Expires: time.Now().Unix() - 100})
	message := tokenHeader + "." + base64.RawURLEncoding.EncodeToString(expiredPayload)
	expired := message + "." + base64.RawURLEncoding.EncodeToString(tokens.sign(message))
	for _, tc := range []struct {
		name, token string
		want        bool
	}{
		{"valid", valid, true}, {"tampered", valid + "x", false}, {"expired", expired, false},
		{"unsigned", tokenHeader + ".e30.", false}, {"missing", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Authorization", "Bearer "+tc.token)
			got, err := tokens.Authenticate(r)
			if (err == nil) != tc.want || (err == nil && got != id) {
				t.Fatalf("id=%v err=%v", got, err)
			}
		})
	}
	if _, err := NewTokens("short"); err == nil {
		t.Fatal("accepted short key")
	}
}
