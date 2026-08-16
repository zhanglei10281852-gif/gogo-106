// Command tessera is a study tool for robust mesh generation.
package main

import (
	"os"

	"Tessera/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
