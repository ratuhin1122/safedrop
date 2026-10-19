package share

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"safedrop/internal/db"
	"safedrop/internal/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrGone         = errors.New("share link is unavailable")
	ErrExpired      = fmt.Errorf("%w: expired", ErrGone)
	ErrLimitReached = fmt.Errorf("%w: download limit reached", ErrGone)
	ErrPassword     = errors.New("share password required or incorrect")
	ErrInvalid      = errors.New("invalid share settings")
	ErrNotFound     = errors.New("owned ready file not found")
)

type Repository interface {
	Create(context.Context, db.CreateShareLinkParams) (db.ShareLink, error)
	Lookup(context.Context, []byte) (db.ShareLink, error)
	File(context.Context, pgtype.UUID) (db.File, error)
	Consume(context.Context, []byte) (db.ShareLink, error)
}

type Service struct {
	Repository Repository
	Storage    storage.Store
	Presign    bool
}

type CreateOptions struct {
	ExpiresAt    time.Time
	Password     string
	MaxDownloads *int64
}

type Created struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *Service) Create(ctx context.Context, owner, file pgtype.UUID, options CreateOptions) (Created, error) {
	if !owner.Valid || !file.Valid || !options.ExpiresAt.After(time.Now()) || len(options.Password) > 72 ||
		(options.MaxDownloads != nil && *options.MaxDownloads < 1) {
		return Created{}, ErrInvalid
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return Created{}, err
	}
	hash := sha256.Sum256(raw[:])
	params := db.CreateShareLinkParams{TokenHash: hash[:], FileID: file, OwnerID: owner,
		ExpiresAt: pgtype.Timestamptz{Time: options.ExpiresAt, Valid: true}}
	if options.MaxDownloads != nil {
		params.MaxDownloads = pgtype.Int8{Int64: *options.MaxDownloads, Valid: true}
	}
	if options.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(options.Password), 12)
		if err != nil {
			return Created{}, err
		}
		params.PasswordHash = pgtype.Text{String: string(hash), Valid: true}
	}
	row, err := s.Repository.Create(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return Created{}, ErrNotFound
	}
	if err != nil {
		return Created{}, err
	}
	return Created{URL: "/s/" + base64.RawURLEncoding.EncodeToString(raw[:]), ExpiresAt: row.ExpiresAt.Time}, nil
}

func tokenHash(token string) ([]byte, error) {
	if len(token) != 22 {
		return nil, ErrGone
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != 16 || base64.RawURLEncoding.EncodeToString(raw) != token {
		return nil, ErrGone
	}
	hash := sha256.Sum256(raw)
	return hash[:], nil
}
