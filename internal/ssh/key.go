package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// FindIdentityKey finds the SSH identity key file
func FindIdentityKey(explicitPath string) (string, error) {
	if explicitPath != "" {
		return explicitPath, nil
	}

	// Default keys to check
	defaultKeys := []string{
		"~/.ssh/id_ed25519",
		"~/.ssh/id_rsa",
		"~/.ssh/id_ecdsa",
	}

	for _, keyPath := range defaultKeys {
		path := ExpandTilde(keyPath)
		if fileExists(path) {
			return path, nil
		}
	}

	return "", fmt.Errorf("no SSH identity key found")
}

// GetCertificatePath returns the certificate path for a given key
func GetCertificatePath(keyPath string) string {
	return keyPath + "-cert.pub"
}

// IsCertificateValid checks if a certificate exists and is valid
func IsCertificateValid(certPath string, principal string) (bool, error) {
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return false, nil
	}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(certData)
	if err != nil {
		return false, fmt.Errorf("failed to parse certificate: %w", err)
	}

	cert, ok := pubKey.(*ssh.Certificate)
	if !ok {
		return false, nil
	}

	// Check certificate type
	if cert.CertType != ssh.UserCert {
		return false, nil
	}

	// Check expiration
	now := uint64(time.Now().Unix())
	if cert.ValidBefore != ssh.CertTimeInfinity && now > cert.ValidBefore {
		return false, nil
	}

	// Check principal
	for _, p := range cert.ValidPrincipals {
		if p == principal {
			return true, nil
		}
	}

	return false, nil
}

// LoadPublicKey loads a public key from a file
func LoadPublicKey(path string) (ssh.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	key, _, _, _, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}

	return key, nil
}

// ExpandTilde expands ~ to home directory
func ExpandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
