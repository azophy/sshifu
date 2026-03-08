package cert

import (
	"crypto/rand"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

// SignUserKey signs a user certificate
// principal is typically the GitHub username
// ttl is the certificate validity duration
// extensions are the permitted capabilities (e.g., permit-pty)
func SignUserKey(ca ssh.Signer, userKey ssh.PublicKey, principal string, ttl time.Duration, extensions map[string]bool) ([]byte, error) {
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
			"permit-agent-forwarding": true,
			"permit-x11-forwarding":  true,
		}
	}

	// Convert map[string]bool to map[string]string for extensions
	certExtensions := make(map[string]string)
	for k, v := range extensions {
		if v {
			certExtensions[k] = ""
		}
	}
	cert.Permissions.Extensions = certExtensions

	if err := cert.SignCert(rand.Reader, ca); err != nil {
		return nil, fmt.Errorf("failed to sign user certificate: %w", err)
	}

	return ssh.MarshalAuthorizedKey(cert), nil
}
