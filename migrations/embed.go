package migrations

import "embed"

// Files contains the application schema used by the one-shot migration service.
//
//go:embed *.sql
var Files embed.FS
