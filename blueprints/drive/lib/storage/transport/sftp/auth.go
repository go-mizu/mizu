// File: lib/storage/transport/sftp/auth.go
package sftp

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// PublicKeyAuthFromFile creates a PublicKeyCallback from an authorized_keys file.
// The file format is the standard OpenSSH authorized_keys format.
func PublicKeyAuthFromFile(path string) (func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error), error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("sftp: read authorized_keys: %w", err)
	}

	return PublicKeyAuthFromBytes(data)
}

// PublicKeyAuthFromBytes creates a PublicKeyCallback from authorized_keys content.
func PublicKeyAuthFromBytes(data []byte) (func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error), error) {
	// Parse all keys
	type authorizedKey struct {
		key     ssh.PublicKey
		comment string
	}

	var keys []authorizedKey

	for len(data) > 0 {
		key, comment, _, rest, err := ssh.ParseAuthorizedKey(data)
		if err != nil {
			// Skip invalid lines
			idx := bytes.IndexByte(data, '\n')
			if idx == -1 {
				break
			}
			data = rest
			continue
		}
		keys = append(keys, authorizedKey{key: key, comment: comment})
		data = rest
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("sftp: no valid keys found in authorized_keys")
	}

	return func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		keyBytes := key.Marshal()

		for _, ak := range keys {
			if bytes.Equal(ak.key.Marshal(), keyBytes) {
				return &ssh.Permissions{
					Extensions: map[string]string{
						"pubkey-comment": ak.comment,
					},
				}, nil
			}
		}

		return nil, fmt.Errorf("unknown public key for %s", conn.User())
	}, nil
}

// PublicKeyAuthFromMap creates a PublicKeyCallback from a map of username to authorized keys.
func PublicKeyAuthFromMap(userKeys map[string][]ssh.PublicKey) func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
	return func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		username := conn.User()
		authorizedKeys, ok := userKeys[username]
		if !ok {
			return nil, fmt.Errorf("unknown user: %s", username)
		}

		keyBytes := key.Marshal()
		for _, ak := range authorizedKeys {
			if bytes.Equal(ak.Marshal(), keyBytes) {
				return &ssh.Permissions{}, nil
			}
		}

		return nil, fmt.Errorf("unknown public key for %s", username)
	}
}

// PasswordAuthFromMap creates a PasswordCallback from a map of username to password.
// Passwords should be stored as hashes in production; this is for testing only.
func PasswordAuthFromMap(userPasswords map[string]string) func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) {
	return func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		username := conn.User()
		expected, ok := userPasswords[username]
		if !ok {
			return nil, fmt.Errorf("unknown user: %s", username)
		}

		if subtle.ConstantTimeCompare([]byte(expected), password) != 1 {
			return nil, fmt.Errorf("invalid password for %s", username)
		}

		return &ssh.Permissions{}, nil
	}
}

// GenerateHostKey generates a new ED25519 host key for testing.
func GenerateHostKey() (ssh.Signer, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.NewSignerFromSigner(&ed25519CryptoSigner{pub: pub, priv: priv})
	if err != nil {
		return nil, err
	}

	return signer, nil
}

// ed25519CryptoSigner implements crypto.Signer for ED25519 keys.
type ed25519CryptoSigner struct {
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

func (s *ed25519CryptoSigner) Public() crypto.PublicKey {
	return s.pub
}

func (s *ed25519CryptoSigner) Sign(_ io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
	return ed25519.Sign(s.priv, digest), nil
}

// ParseAuthorizedKeysLine parses a single line from an authorized_keys file.
func ParseAuthorizedKeysLine(line string) (ssh.PublicKey, string, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, "", fmt.Errorf("empty or comment line")
	}

	key, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
	if err != nil {
		return nil, "", err
	}

	return key, comment, nil
}
