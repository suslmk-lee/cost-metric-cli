package main

import (
	"os"

	"github.com/minkyulee/cost-metric-cli/pkg/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}