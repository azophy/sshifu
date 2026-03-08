package cert

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// CA represents a Certificate Authority for SSH
type CA struct {
	signer ssh.Signer
	pubKey ssh.PublicKey
}

// LoadCA loads a CA from a private key file
func LoadCA(privateKeyPath string) (*CA, error) {
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA private key: %w", err)
	}

	return &CA{
		signer: signer,
		pubKey: signer.PublicKey(),
	}, nil
}

// GenerateCA generates a new CA keypair
func GenerateCA(privateKeyPath, publicKeyPath string) error {
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

// SignUserKey signs a user certificate using the CA
func (ca *CA) SignUserKey(userKey ssh.PublicKey, principal string, ttl time.Duration, extensions map[string]bool) ([]byte, error) {
	return SignUserKey(ca.signer, userKey, principal, ttl, extensions)
}

// SignHostKey signs a host certificate using the CA
func (ca *CA) SignHostKey(hostKey ssh.PublicKey, principals []string, ttl time.Duration) ([]byte, error) {
	return SignHostKey(ca.signer, hostKey, principals, ttl)
}

// PublicKey returns the CA public key
func (ca *CA) PublicKey() ssh.PublicKey {
	return ca.pubKey
}

// Signer returns the CA signer
func (ca *CA) Signer() ssh.Signer {
	return ca.signer
}
