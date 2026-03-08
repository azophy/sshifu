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

// SignUserKey signs a user certificate
func (ca *CA) SignUserKey(userKey ssh.PublicKey, principal string, ttl time.Duration, extensions map[string]bool) ([]byte, error) {
	cert := &ssh.Certificate{
		Key:             userKey,
		CertType:        ssh.UserCert,
		KeyId:           principal,
		ValidPrincipals: []string{principal},
		ValidBefore:     ssh.CertTimeInfinity,
		ValidAfter:      uint64(time.Now().Unix()),
	}

	if ttl > 0 {
		cert.ValidBefore = uint64(time.Now().Add(ttl).Unix())
	}

	// Set default extensions if not specified
	if len(extensions) == 0 {
		extensions = map[string]bool{
			"permit-pty":             true,
			"permit-port-forwarding": true,
		}
	}

	// Convert map[string]bool to map[string]string
	certExtensions := make(map[string]string)
	for k, v := range extensions {
		if v {
			certExtensions[k] = ""
		}
	}
	cert.Permissions.Extensions = certExtensions

	if err := cert.SignCert(rand.Reader, ca.signer); err != nil {
		return nil, fmt.Errorf("failed to sign certificate: %w", err)
	}

	return ssh.MarshalAuthorizedKey(cert), nil
}

// SignHostKey signs a host certificate
func (ca *CA) SignHostKey(hostKey ssh.PublicKey, principals []string, ttl time.Duration) ([]byte, error) {
	cert := &ssh.Certificate{
		Key:             hostKey,
		CertType:        ssh.HostCert,
		KeyId:           "host",
		ValidPrincipals: principals,
		ValidBefore:     ssh.CertTimeInfinity,
		ValidAfter:      uint64(time.Now().Unix()),
	}

	if ttl > 0 {
		cert.ValidBefore = uint64(time.Now().Add(ttl).Unix())
	}

	if err := cert.SignCert(rand.Reader, ca.signer); err != nil {
		return nil, fmt.Errorf("failed to sign host certificate: %w", err)
	}

	return ssh.MarshalAuthorizedKey(cert), nil
}

// PublicKey returns the CA public key
func (ca *CA) PublicKey() ssh.PublicKey {
	return ca.pubKey
}
