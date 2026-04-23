package cli

import (
	urfave "github.com/urfave/cli/v2"
)

// Version is overridden at build time via -ldflags "-X .../internal/cli.Version=..."
var Version = "0.1.0-dev"

func NewApp() *urfave.App {
	return &urfave.App{
		Name:    "nexus3-cli",
		Usage:   "CLI for Sonatype Nexus Repository 3 (Docker v2)",
		Version: Version,
		Commands: []*urfave.Command{
			configureCommand(),
			imageCommand(),
		},
	}
}