package migrations

import "embed"

//go:embed *.migration.sql
var Fs = embed.FS{}
