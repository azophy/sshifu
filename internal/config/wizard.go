package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"golang.org/x/crypto/ssh"
)

// WizardResult represents the result of the setup wizard
type WizardResult struct {
	Config       *Config
	GeneratedCA  bool
	CAKeyPath    string
	CAPublicPath string
}

// RunWizard runs an interactive setup wizard
func RunWizard(configPath string) (*WizardResult, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🔐 Sshifu Server Setup Wizard")
	fmt.Println("=============================")
	fmt.Println()
	fmt.Println("This wizard will help you configure sshifu-server.")
	fmt.Println()

	// Server public URL
	fmt.Print("Server public URL (e.g., https://auth.example.com): ")
	publicURL, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read public URL: %w", err)
	}
	publicURL = strings.TrimSpace(publicURL)
	if publicURL == "" {
		return nil, fmt.Errorf("public URL cannot be empty")
	}

	// CA key path
	fmt.Print("CA private key path (default: ./ca): ")
	caKeyPath, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read CA key path: %w", err)
	}
	caKeyPath = strings.TrimSpace(caKeyPath)
	if caKeyPath == "" {
		caKeyPath = "./ca"
	}

	// Derive public key path from private key path
	caPublicPath := caKeyPath + ".pub"

	// Check if CA keys already exist
	generatedCA := false
	if !fileExists(caKeyPath) {
		fmt.Println()
		fmt.Println("CA keypair not found. Generating new CA keys...")
		if err := GenerateCAKeys(caKeyPath, caPublicPath); err != nil {
			return nil, fmt.Errorf("failed to generate CA keys: %w", err)
		}
		fmt.Println("✓ CA keypair generated successfully")
		generatedCA = true
	} else {
		fmt.Println()
		fmt.Println("✓ CA keypair already exists")
	}

	// GitHub OAuth configuration
	fmt.Println()
	fmt.Println("GitHub OAuth Configuration:")
	fmt.Println("You'll need to create a GitHub OAuth app at: https://github.com/settings/developers")
	fmt.Println()

	fmt.Print("GitHub Client ID: ")
	clientID, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read client ID: %w", err)
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, fmt.Errorf("client ID cannot be empty")
	}

	fmt.Print("GitHub Client Secret: ")
	clientSecret, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read client secret: %w", err)
	}
	clientSecret = strings.TrimSpace(clientSecret)
	if clientSecret == "" {
		return nil, fmt.Errorf("client secret cannot be empty")
	}

	fmt.Print("Allowed GitHub Organization: ")
	allowedOrg, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read allowed org: %w", err)
	}
	allowedOrg = strings.TrimSpace(allowedOrg)
	if allowedOrg == "" {
		return nil, fmt.Errorf("allowed organization cannot be empty")
	}

	// Create configuration
	cfg := &Config{}
	cfg.Server.Listen = ":8080"
	cfg.Server.PublicURL = publicURL
	cfg.CA.PrivateKey = caKeyPath
	cfg.CA.PublicKey = caPublicPath
	cfg.Cert.TTL = "8h"
	cfg.Cert.Extensions = map[string]bool{
		"permit-pty":             true,
		"permit-port-forwarding": true,
	}
	cfg.Auth.Providers = []Provider{
		{
			Name:       "github",
			Type:       "github",
			ClientID:   clientID,
			ClientSecret: clientSecret,
			AllowedOrg: allowedOrg,
		},
	}

	// Save configuration
	configDir := filepath.Dir(configPath)
	if configDir != "" && configDir != "." {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	if err := Save(cfg, configPath); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Configuration saved to", configPath)
	fmt.Println()
	fmt.Println("Setup complete! You can now start sshifu-server.")

	return &WizardResult{
		Config:       cfg,
		GeneratedCA:  generatedCA,
		CAKeyPath:    caKeyPath,
		CAPublicPath: caPublicPath,
	}, nil
}

// Save saves the configuration to a YAML file
func Save(cfg *Config, path string) error {
	data, err := Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Marshal marshals the configuration to YAML
func Marshal(cfg *Config) ([]byte, error) {
	return yaml.Marshal(cfg)
}

// GenerateCAKeys generates a new CA keypair
func GenerateCAKeys(privateKeyPath, publicKeyPath string) error {
	// Generate ED25519 key
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Convert to SSH private key
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}

	// Marshal private key to PEM
	privKeyPEM, err := ssh.MarshalPrivateKey(priv, "sshifu-ca")
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Write private key
	if err := os.WriteFile(privateKeyPath, pem.EncodeToMemory(privKeyPEM), 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Write public key
	pubKey := ssh.MarshalAuthorizedKey(signer.PublicKey())
	if err := os.WriteFile(publicKeyPath, []byte(pubKey), 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

// fileExists checks if a file exists (not a directory)
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
