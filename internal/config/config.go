package config

import (
	"fmt"
	"os"
	"time"

	"github.com/goccy/go-yaml"
)

// Config represents the server configuration
type Config struct {
	Server struct {
		Listen    string `yaml:"listen"`
		PublicURL string `yaml:"public_url"`
	} `yaml:"server"`

	CA struct {
		PrivateKey string `yaml:"private_key"`
		PublicKey  string `yaml:"public_key"`
	} `yaml:"ca"`

	Cert struct {
		TTL        string            `yaml:"ttl"`
		Extensions map[string]bool   `yaml:"extensions"`
	} `yaml:"cert"`

	Auth struct {
		Providers []Provider `yaml:"providers"`
	} `yaml:"auth"`
}

// Provider represents an OAuth provider configuration
type Provider struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	ClientID    string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	AllowedOrg  string `yaml:"allowed_org,omitempty"`
	Issuer      string `yaml:"issuer,omitempty"`
	PrincipalOAuthFieldName string `yaml:"principal_oauth_field_name,omitempty"`
}

// DefaultTTL returns the default certificate TTL
func (c *Config) DefaultTTL() time.Duration {
	if c.Cert.TTL == "" {
		return 8 * time.Hour
	}
	d, err := time.ParseDuration(c.Cert.TTL)
	if err != nil {
		return 8 * time.Hour
	}
	return d
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":8080"
	}
	if cfg.CA.PrivateKey == "" {
		cfg.CA.PrivateKey = "./ca"
	}
	if cfg.CA.PublicKey == "" {
		cfg.CA.PublicKey = "./ca.pub"
	}
	if cfg.Cert.TTL == "" {
		cfg.Cert.TTL = "8h"
	}

	return &cfg, nil
}

// Exists checks if a config file exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
