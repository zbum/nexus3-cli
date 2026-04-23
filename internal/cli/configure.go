package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	urfave "github.com/urfave/cli/v2"
	"golang.org/x/term"

	"github.com/zbum/nexus3-cli/internal/config"
)

func configureCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "configure",
		Usage: "Configure Nexus credentials (saved to ~/.nexus-cli)",
		Action: func(c *urfave.Context) error {
			existing, _ := config.Load()
			reader := bufio.NewReader(os.Stdin)

			host := prompt(reader, "Nexus Host (e.g. http://nexus:8081)", fallback(existing, func(c *config.Config) string { return c.Host }))
			user := prompt(reader, "Nexus Username", fallback(existing, func(c *config.Config) string { return c.Username }))
			pass, err := promptSecret("Nexus Password", fallback(existing, func(c *config.Config) string { return c.Password }))
			if err != nil {
				return err
			}
			repo := prompt(reader, "Nexus Docker Repository name", fallback(existing, func(c *config.Config) string { return c.Repository }))

			cfg := &config.Config{
				Host:       host,
				Username:   user,
				Password:   pass,
				Repository: repo,
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			p, _ := config.Path()
			fmt.Fprintf(c.App.Writer, "Saved %s\n", p)
			return nil
		},
	}
}

func fallback(existing *config.Config, pick func(*config.Config) string) string {
	if existing == nil {
		return ""
	}
	return pick(existing)
}

func prompt(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return def
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func promptSecret(label, def string) (string, error) {
	if def != "" {
		fmt.Printf("%s [****]: ", label)
	} else {
		fmt.Printf("%s: ", label)
	}
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		r := bufio.NewReader(os.Stdin)
		line, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			return def, nil
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return def, nil
		}
		return line, nil
	}
	b, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return def, nil
	}
	return s, nil
}