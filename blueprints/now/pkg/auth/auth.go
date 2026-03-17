package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// Identity holds an actor's keypair and identifiers.
type Identity struct {
	Actor       string
	Fingerprint string
	PublicKey   ed25519.PublicKey
	PrivateKey  ed25519.PrivateKey
}

// SignedRequest carries a signed operation request.
type SignedRequest struct {
	Actor       string
	Fingerprint string
	PublicKey   []byte
	Payload     []byte
	Signature   []byte
	Nonce       string
	Timestamp   int64
}

// VerifiedActor is produced only by Verify and proves the actor's identity.
type VerifiedActor struct {
	Actor       string
	Fingerprint string
}

// GenerateIdentity creates a new ed25519 keypair for the given actor name.
func GenerateIdentity(actor string) (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &Identity{
		Actor:       actor,
		Fingerprint: Fingerprint(pub),
		PublicKey:   pub,
		PrivateKey:  priv,
	}, nil
}

// Fingerprint returns the first 16 hex chars of sha256(publicKey).
func Fingerprint(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return hex.EncodeToString(h[:])[:16]
}
