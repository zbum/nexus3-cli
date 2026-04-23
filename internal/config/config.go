package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const fileName = ".nexus-cli"

type Config struct {
	Host       string
	Username   string
	Password   string
	Repository string
}

func Path() (string, error) {
	if v := os.Getenv("NEXUS_CLI_CONFIG"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, fileName), nil
}

func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found at %s; run `nexus3-cli configure`", p)
		}
		return nil, err
	}
	defer f.Close()

	c := &Config{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"`)
		switch key {
		case "nexus_host":
			c.Host = val
		case "nexus_username":
			c.Username = val
		case "nexus_password":
			c.Password = val
		case "nexus_repository":
			c.Repository = val
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if c.Host == "" {
		return nil, errors.New("nexus_host not set; run `nexus3-cli configure`")
	}
	return c, nil
}

func Save(c *Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	content := fmt.Sprintf(
		"nexus_host = %q\nnexus_username = %q\nnexus_password = %q\nnexus_repository = %q\n",
		c.Host, c.Username, c.Password, c.Repository,
	)
	return os.WriteFile(p, []byte(content), 0o600)
}