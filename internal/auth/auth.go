package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

var ErrUnauthorized = errors.New("authentication required")

type Authenticator interface {
	Authenticate(*http.Request) (pgtype.UUID, error)
}

type Deny struct{}

func (Deny) Authenticate(*http.Request) (pgtype.UUID, error) {
	return pgtype.UUID{}, ErrUnauthorized
}

// Stub authenticates one explicitly configured development user. It does not
// create users or implement production authentication. Empty settings deny all.
type Stub struct {
	userID pgtype.UUID
	token  [32]byte
}

func NewStub(userID, token string) (Authenticator, error) {
	if userID == "" && token == "" {
		return Deny{}, nil
	}
	var id pgtype.UUID
	if err := id.Scan(userID); err != nil || !id.Valid || id.Bytes == [16]byte{} {
		return nil, errors.New("STUB_AUTH_USER_ID must be a nonzero UUID")
	}
	if len(token) < 32 || strings.ContainsAny(token, " \t\r\n") {
		return nil, errors.New("STUB_AUTH_TOKEN must contain at least 32 non-whitespace bytes")
	}
	return &Stub{userID: id, token: sha256.Sum256([]byte(token))}, nil
}

func (s *Stub) Authenticate(r *http.Request) (pgtype.UUID, error) {
	values := r.Header.Values("Authorization")
	if len(values) != 1 {
		return pgtype.UUID{}, ErrUnauthorized
	}
	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return pgtype.UUID{}, ErrUnauthorized
	}
	digest := sha256.Sum256([]byte(parts[1]))
	if subtle.ConstantTimeCompare(digest[:], s.token[:]) != 1 {
		return pgtype.UUID{}, ErrUnauthorized
	}
	return s.userID, nil
}
