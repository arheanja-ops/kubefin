// Command kubefin es una CLI para diagnóstico de EKS y análisis de costos AWS,
// con superficies CLI, servidor MCP y agente de IA. Ver docs/ para el diseño.
package main

import (
	"fmt"
	"os"

	"github.com/arheanja-ops/kubefin/internal/cli"
)

// version se sobrescribe en build vía -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := cli.NewRootCmd(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
