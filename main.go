package main

import (
	"fmt"
	"os"

	"github.com/zbum/nexus3-cli/internal/cli"
)

func main() {
	if err := cli.NewApp().Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}