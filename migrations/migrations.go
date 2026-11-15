// Package migrations expõe as migrações SQL embutidas no binário.
package migrations

import "embed"

// FS contém os arquivos de migração versionados.
//
//go:embed *.sql
var FS embed.FS
