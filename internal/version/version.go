package version

import "fmt"

// Valores substituídos em tempo de build por meio de ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String retorna uma descrição legível da versão em execução.
func String() string {
	return fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date)
}
