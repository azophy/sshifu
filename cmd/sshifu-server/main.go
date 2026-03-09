package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/azophy/sshifu/internal/api"
	"github.com/azophy/sshifu/internal/cert"
	"github.com/azophy/sshifu/internal/config"
	"github.com/azophy/sshifu/internal/oauth"
	"github.com/azophy/sshifu/internal/session"
)

var (
	version    = "0.0.0-dev"
	configPath = "config.yml"
)

func main() {
	// Handle special commands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "-help", "--help", "help":
			printUsage()
			os.Exit(0)
		case "-v", "-version", "--version", "version":
			printVersion()
			os.Exit(0)
		}
	}

	fmt.Println("🔐 Sshifu Server - SSH Certificate Authority and OAuth Gateway")
	fmt.Println()

	// Check if config exists, if not run wizard
	if !config.Exists(configPath) {
		fmt.Println("Configuration not found. Starting setup wizard...")
		fmt.Println()

		result, err := config.RunWizard(configPath)
		if err != nil {
			log.Fatalf("Setup wizard failed: %v", err)
		}

		if result.GeneratedCA {
			fmt.Printf("CA private key: %s\n", result.CAKeyPath)
			fmt.Printf("CA public key: %s\n", result.CAPublicPath)
		}
		fmt.Println()
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Configuration loaded from %s\n", configPath)
	fmt.Printf("Server will listen on: %s\n", cfg.Server.Listen)
	fmt.Printf("Public URL: %s\n", cfg.Server.PublicURL)
	fmt.Printf("CA private key: %s\n", cfg.CA.PrivateKey)
	fmt.Printf("CA public key: %s\n", cfg.CA.PublicKey)
	fmt.Printf("Certificate TTL: %s\n", cfg.Cert.TTL)

	if len(cfg.Auth.Providers) > 0 {
		fmt.Printf("OAuth providers: %d configured\n", len(cfg.Auth.Providers))
		for _, p := range cfg.Auth.Providers {
			fmt.Printf("  - %s (%s)\n", p.Name, p.Type)
		}
	}

	fmt.Println()

	// Load CA private key
	ca, err := cert.LoadCA(cfg.CA.PrivateKey)
	if err != nil {
		log.Fatalf("Failed to load CA private key: %v", err)
	}
	fmt.Println("✓ CA loaded successfully")

	// Initialize session store
	sessionStore := session.NewStore(15 * time.Minute)
	fmt.Println("✓ Session store initialized")

	// Find and initialize OAuth provider
	var oauthProvider oauth.Provider
	for _, p := range cfg.Auth.Providers {
		if p.Type == "github" {
			oauthProvider = oauth.NewGitHubProvider(
				p.ClientID,
				p.ClientSecret,
				cfg.Server.PublicURL+"/oauth/callback",
				p.AllowedOrg,
			)
			fmt.Printf("✓ OAuth provider initialized: %s\n", p.Name)
			break
		}
	}

	if oauthProvider == nil {
		log.Fatal("No supported OAuth provider configured")
	}

	// Load CA public key
	caPubKeyBytes, err := os.ReadFile(cfg.CA.PublicKey)
	if err != nil {
		log.Fatalf("Failed to read CA public key: %v", err)
	}
	caPubKey := strings.TrimSpace(string(caPubKeyBytes))

	// Prepare handler config
	handlerCfg := &api.Config{
		TTL:        cfg.DefaultTTL(),
		Extensions: cfg.Cert.Extensions,
	}

	// Initialize API handler
	handler, err := api.NewHandler(sessionStore, oauthProvider, ca.Signer(), handlerCfg, cfg.Server.PublicURL)
	if err != nil {
		log.Fatalf("Failed to initialize API handler: %v", err)
	}

	// Setup routes
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/v1/login/start", handler.LoginStart)
	mux.HandleFunc("/api/v1/login/status/", handler.LoginStatus)
	mux.HandleFunc("/api/v1/ca/pub", func(w http.ResponseWriter, r *http.Request) {
		handler.CAPublicKey(w, r, caPubKey)
	})
	mux.HandleFunc("/api/v1/sign/user", handler.SignUserCertificate)
	mux.HandleFunc("/api/v1/sign/host", handler.SignHostCertificate)

	// OAuth routes
	mux.HandleFunc("/oauth/callback", handler.OAuthCallback)
	mux.HandleFunc("/oauth/github/", handler.OAuthInit)

	// Login page
	mux.HandleFunc("/login/", handler.Login)

	fmt.Println("✓ Routes configured")
	fmt.Println()
	fmt.Println("🚀 Server starting...")
	fmt.Println()

	// Start server
	if err := http.ListenAndServe(cfg.Server.Listen, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func printVersion() {
	fmt.Printf("sshifu-server version %s\n", version)
}

func printUsage() {
	fmt.Printf("sshifu-server version %s\n", version)
	fmt.Println()
	fmt.Println("Usage: sshifu-server [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  help, -h, --help     Show this help message")
	fmt.Println("  version, -v, --version  Show version information")
	fmt.Println()
	fmt.Println("Description:")
	fmt.Println("  SSH Certificate Authority and OAuth Gateway")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  Configuration file: config.yml")
	fmt.Println("  If no configuration exists, a setup wizard will run.")
}
