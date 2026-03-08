package main

import (
	"fmt"
	"log"
	"os"

	"github.com/azophy/sshifu/internal/config"
)

const configPath = "config.yml"

func main() {
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
	fmt.Println("Server starting... (implementation in progress)")

	// TODO: Start HTTP server with API endpoints
	// This will be implemented in Milestone 6

	if len(os.Args) > 1 {
		fmt.Printf("Arguments: %v\n", os.Args[1:])
	}
}
