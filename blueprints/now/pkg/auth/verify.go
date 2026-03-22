package auth

import (
	"context"
	"crypto/ed25519"
	"errors"
	"time"
)

// DefaultTimestampWindow is the maximum age of a signed request.
const DefaultTimestampWindow = 30 * time.Second

// Verify checks signature, timestamp, nonce, and key binding.
// Returns a VerifiedActor only if all checks pass.
func Verify(ctx context.Context, req *SignedRequest, keys KeyStore, nonces NonceStore) (*VerifiedActor, error) {
	// 1. Check timestamp window.
	now := time.Now().UnixMilli()
	if now-req.Timestamp > DefaultTimestampWindow.Milliseconds() {
		return nil, errors.New("request expired")
	}

	// 2. Check nonce uniqueness.
	if err := nonces.Check(req.Nonce, req.Timestamp); err != nil {
		return nil, err
	}

	// 3. Verify ed25519 signature.
	pubKey := ed25519.PublicKey(req.PublicKey)
	if len(pubKey) != ed25519.PublicKeySize {
		return nil, errors.New("invalid signature")
	}
	if !ed25519.Verify(pubKey, req.Payload, req.Signature) {
		return nil, errors.New("invalid signature")
	}

	// 4. Derive fingerprint and verify it matches the claim.
	fp := Fingerprint(pubKey)
	if fp != req.Fingerprint {
		return nil, errors.New("fingerprint mismatch")
	}

	// 5. Register or verify key binding.
	if err := keys.Register(ctx, req.Actor, req.PublicKey); err != nil {
		return nil, err
	}

	return &VerifiedActor{
		Actor:       req.Actor,
		Fingerprint: fp,
	}, nil
}
