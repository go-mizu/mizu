package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Sign creates a SignedRequest for the given operation and parameters.
func Sign(id *Identity, operation string, params map[string]string) (*SignedRequest, error) {
	nonce, err := generateNonce()
	if err != nil {
		return nil, err
	}

	ts := time.Now().UnixMilli()

	// Include actor and fingerprint in the signed payload so they
	// cannot be swapped without invalidating the signature.
	all := make(map[string]string)
	for k, v := range params {
		if v != "" {
			all[k] = v
		}
	}
	all["actor"] = id.Actor
	all["fingerprint"] = id.Fingerprint
	all["nonce"] = nonce
	all["ts"] = fmt.Sprintf("%d", ts)

	payload := canonical(operation, all)
	sig := ed25519.Sign(id.PrivateKey, []byte(payload))

	return &SignedRequest{
		Actor:       id.Actor,
		Fingerprint: id.Fingerprint,
		PublicKey:   []byte(id.PublicKey),
		Payload:     []byte(payload),
		Signature:   sig,
		Nonce:       nonce,
		Timestamp:   ts,
	}, nil
}

// canonical builds a deterministic string from operation and sorted params.
func canonical(operation string, params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys)+1)
	parts = append(parts, operation)
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	return strings.Join(parts, ":")
}

// generateNonce returns 16 random bytes hex-encoded (32 chars).
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
