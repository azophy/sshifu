package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"bufio"
	"fmt"
	"os"
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

	// Choose OAuth provider type
	fmt.Println()
	fmt.Println("Select OAuth Provider Type:")
	fmt.Println("  1) GitHub (GitHub organization-based authentication)")
	fmt.Println("  2) Generic OIDC (OpenID Connect compatible providers)")
	fmt.Println()
	fmt.Print("Choose provider type [1-2] (default: 1): ")
	providerChoice, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read provider choice: %w", err)
	}
	providerChoice = strings.TrimSpace(providerChoice)
	if providerChoice == "" {
		providerChoice = "1"
	}

	var primaryProvider Provider
	var secondaryProviderType string

	if providerChoice == "2" || providerChoice == "oidc" {
		// OIDC configuration
		fmt.Println()
		fmt.Println("Generic OIDC Configuration:")
		fmt.Println()

		fmt.Print("OIDC Issuer URL (e.g., https://accounts.google.com): ")
		issuer, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read issuer URL: %w", err)
		}
		issuer = strings.TrimSpace(issuer)
		if issuer == "" {
			return nil, fmt.Errorf("issuer URL cannot be empty")
		}

		fmt.Print("OIDC Client ID: ")
		clientID, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read client ID: %w", err)
		}
		clientID = strings.TrimSpace(clientID)
		if clientID == "" {
			return nil, fmt.Errorf("client ID cannot be empty")
		}

		fmt.Print("OIDC Client Secret: ")
		clientSecret, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read client secret: %w", err)
		}
		clientSecret = strings.TrimSpace(clientSecret)
		if clientSecret == "" {
			return nil, fmt.Errorf("client secret cannot be empty")
		}

		fmt.Print("Principal OAuth Field Name (e.g., preferred_username, email): ")
		principalField, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read principal field name: %w", err)
		}
		principalField = strings.TrimSpace(principalField)
		if principalField == "" {
			principalField = "preferred_username"
		}

		primaryProvider = Provider{
			Name:        "oidc",
			Type:        "oidc",
			ClientID:    clientID,
			ClientSecret: clientSecret,
			Issuer:      issuer,
			PrincipalOAuthFieldName: principalField,
		}
		secondaryProviderType = "github"
	} else {
		// GitHub OAuth configuration (default)
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

		primaryProvider = Provider{
			Name:       "github",
			Type:       "github",
			ClientID:   clientID,
			ClientSecret: clientSecret,
			AllowedOrg: allowedOrg,
		}
		secondaryProviderType = "oidc"
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
	cfg.Auth.Providers = []Provider{primaryProvider}

	// Save configuration with commented alternative provider
	if err := SaveWithCommentedAlternative(cfg, configPath, secondaryProviderType); err != nil {
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

// SaveWithCommentedAlternative saves the configuration with an alternative provider as commented section
func SaveWithCommentedAlternative(cfg *Config, path string, alternativeType string) error {
	var sb strings.Builder

	// Write header
	sb.WriteString("# Sshifu Server Configuration\n")
	sb.WriteString("# Copy this file to config.yml and fill in your values\n\n")

	// Server section
	sb.WriteString("server:\n")
	sb.WriteString(fmt.Sprintf("  listen: %q\n", cfg.Server.Listen))
	sb.WriteString(fmt.Sprintf("  public_url: %s\n", cfg.Server.PublicURL))
	sb.WriteString("\n")

	// CA section
	sb.WriteString("ca:\n")
	sb.WriteString(fmt.Sprintf("  private_key: %s\n", cfg.CA.PrivateKey))
	sb.WriteString(fmt.Sprintf("  public_key: %s\n", cfg.CA.PublicKey))
	sb.WriteString("\n")

	// Cert section
	sb.WriteString("cert:\n")
	sb.WriteString(fmt.Sprintf("  ttl: %s\n", cfg.Cert.TTL))
	sb.WriteString("  extensions:\n")
	for k, v := range cfg.Cert.Extensions {
		sb.WriteString(fmt.Sprintf("    %s: %v\n", k, v))
	}
	sb.WriteString("\n")

	// Auth section with providers
	sb.WriteString("auth:\n")
	sb.WriteString("  providers:\n")

	// Write active provider
	if len(cfg.Auth.Providers) > 0 {
		p := cfg.Auth.Providers[0]
		sb.WriteString("    - name: " + p.Name + "\n")
		sb.WriteString("      type: " + p.Type + "\n")
		sb.WriteString("      client_id: " + p.ClientID + "\n")
		sb.WriteString("      client_secret: " + p.ClientSecret + "\n")
		if p.Type == "github" && p.AllowedOrg != "" {
			sb.WriteString("      allowed_org: " + p.AllowedOrg + "\n")
		}
		if p.Type == "oidc" {
			sb.WriteString("      issuer: " + p.Issuer + "\n")
			if p.PrincipalOAuthFieldName != "" {
				sb.WriteString("      principal_oauth_field_name: " + p.PrincipalOAuthFieldName + "\n")
			}
		}
	}

	// Write commented alternative provider
	sb.WriteString("\n")
	if alternativeType == "github" {
		sb.WriteString("    # Optional: GitHub provider\n")
		sb.WriteString("    # - name: github\n")
		sb.WriteString("    #   type: github\n")
		sb.WriteString("    #   client_id: YOUR_GITHUB_CLIENT_ID\n")
		sb.WriteString("    #   client_secret: YOUR_GITHUB_CLIENT_SECRET\n")
		sb.WriteString("    #   allowed_org: your-github-org\n")
	} else if alternativeType == "oidc" {
		sb.WriteString("    # Optional: OIDC provider\n")
		sb.WriteString("    # - name: oidc\n")
		sb.WriteString("    #   type: oidc\n")
		sb.WriteString("    #   issuer: https://example.com\n")
		sb.WriteString("    #   client_id: YOUR_OIDC_CLIENT_ID\n")
		sb.WriteString("    #   client_secret: YOUR_OIDC_CLIENT_SECRET\n")
		sb.WriteString("    #   principal_oauth_field_name: preferred_username\n")
	}

	// Write file
	if err := os.WriteFile(path, []byte(sb.String()), 0600); err != nil {
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
