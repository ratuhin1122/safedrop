package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStub(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	token := strings.Repeat("s", 32)
	stub, err := NewStub(id, token)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		header     string
		authorized bool
	}{
		{"Bearer " + token, true}, {"bearer " + token, true},
		{"", false}, {"Bearer wrong", false}, {"Basic " + token, false},
		{"Bearer " + token + " extra", false},
	} {
		t.Run(tt.header, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/v1/files", nil)
			r.Header.Set("Authorization", tt.header)
			user, err := stub.Authenticate(r)
			if tt.authorized && (err != nil || user.String() != id) {
				t.Fatalf("user: %v %v", user, err)
			}
			if !tt.authorized && !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("error: %v", err)
			}
		})
	}
	disabled, err := NewStub("", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := disabled.Authenticate(httptest.NewRequest(http.MethodGet, "/", nil)); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("default authentication is open")
	}
	for _, pair := range [][2]string{{"invalid", token}, {id, "short"}, {"", token}, {id, ""}} {
		if _, err := NewStub(pair[0], pair[1]); err == nil {
			t.Fatal("accepted invalid stub settings")
		}
	}
}
