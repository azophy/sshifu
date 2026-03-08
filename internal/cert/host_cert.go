package cert

import (
	"crypto/rand"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

// SignHostKey signs a host certificate
// principals are the hostnames/IPs the certificate is valid for
// ttl is the certificate validity duration
func SignHostKey(ca ssh.Signer, hostKey ssh.PublicKey, principals []string, ttl time.Duration) ([]byte, error) {
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

	if err := cert.SignCert(rand.Reader, ca); err != nil {
		return nil, fmt.Errorf("failed to sign host certificate: %w", err)
	}

	return ssh.MarshalAuthorizedKey(cert), nil
}
