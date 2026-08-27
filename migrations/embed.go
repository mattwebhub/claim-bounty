// Package migrations exposes the immutable database migration set to the
// PostgreSQL adapter without making startup depend on the process directory.
package migrations

import "embed"

// Files contains every versioned Goose migration.
//
//go:embed *.sql
var Files embed.FS
