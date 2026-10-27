// Package migrations embeds versioned SQL migrations into the migration command.
package migrations

import "embed"

// Files contains migrations shipped with this version of the application.
//
//go:embed *.sql
var Files embed.FS
