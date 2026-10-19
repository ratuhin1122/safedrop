package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"safedrop/internal/db"
	"safedrop/internal/observability"
)

var ErrCredentials = errors.New("invalid credentials")
var ErrInput = errors.New("invalid account input")
var ErrExists = errors.New("account already exists")
var ErrBusy = errors.New("authentication busy")

type Accounts struct {
	Queries *db.Queries
	Tokens  *Tokens
	slots   chan struct{}
	dummy   string
}

func NewAccounts(q *db.Queries, tokens *Tokens) *Accounts {
	return &Accounts{Queries: q, Tokens: tokens, slots: make(chan struct{}, 2), dummy: hashPassword("unused dummy password")}
}

func (a *Accounts) Exchange(ctx context.Context, email, password string, register bool) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 || len(password) < 12 || len(password) > 1024 {
		return "", ErrInput
	}
	select {
	case a.slots <- struct{}{}:
		defer func() { <-a.slots }()
	default:
		return "", ErrBusy
	}
	if register {
		u, err := a.Queries.CreateUser(ctx, db.CreateUserParams{Email: email, PasswordHash: hashPassword(password)})
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			return "", ErrExists
		}
		if err != nil {
			return "", err
		}
		observability.SetUserID(ctx, u.ID.String())
		return a.Tokens.Issue(u.ID), nil
	}
	u, err := a.Queries.GetUserByEmail(ctx, email)
	encoded := u.PasswordHash
	if errors.Is(err, pgx.ErrNoRows) {
		encoded = a.dummy
	} else if err != nil {
		return "", err
	}
	valid := checkPassword(encoded, password)
	if err != nil || !valid {
		return "", ErrCredentials
	}
	observability.SetUserID(ctx, u.ID.String())
	return a.Tokens.Issue(u.ID), nil
}
