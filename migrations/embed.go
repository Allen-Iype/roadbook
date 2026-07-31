// Package migrations embeds the goose SQL migrations into the binary, so
// `roadbook migrate` needs no external tool and a deployment cannot drift from
// the schema its binary expects.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
