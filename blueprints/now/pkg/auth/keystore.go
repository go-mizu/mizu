package auth

import "context"

// KeyStore binds actor names to public keys.
type KeyStore interface {
	// Register binds a name to a public key. Fails if the name is already
	// bound to a different key. Idempotent if same name + same key.
	Register(ctx context.Context, actor string, publicKey []byte) error

	// Lookup returns the public key bound to an actor name.
	Lookup(ctx context.Context, actor string) ([]byte, error)
}
