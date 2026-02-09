package x

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TID generation ported from Nitter's src/tid.nim.
// Generates x-client-transaction-id headers required for API access.

const (
	tidKeyword = "obfiowerehiring"
	tidPairsURL = "https://raw.githubusercontent.com/fa0311/x-client-transaction-id-pair-dict/refs/heads/main/pair.json"
	tidCacheTTL = 1 * time.Hour
	// Epoch offset: April 30, 2023 21:00:00 UTC
	tidEpochOffset = 1682924400
)

type tidPair struct {
	AnimationKey string `json:"animationKey"`
	Verification string `json:"verification"`
}

var (
	tidMu         sync.Mutex
	tidPairs      []tidPair
	tidLastFetch  time.Time
	tidHTTPClient = &http.Client{Timeout: 15 * time.Second}
)

func fetchTIDPairs() ([]tidPair, error) {
	tidMu.Lock()
	defer tidMu.Unlock()

	if len(tidPairs) > 0 && time.Since(tidLastFetch) < tidCacheTTL {
		return tidPairs, nil
	}

	resp, err := tidHTTPClient.Get(tidPairsURL)
	if err != nil {
		if len(tidPairs) > 0 {
			return tidPairs, nil // use stale cache
		}
		return nil, fmt.Errorf("fetch TID pairs: %w", err)
	}
	defer resp.Body.Close()

	var pairs []tidPair
	if err := json.NewDecoder(resp.Body).Decode(&pairs); err != nil {
		if len(tidPairs) > 0 {
			return tidPairs, nil
		}
		return nil, fmt.Errorf("parse TID pairs: %w", err)
	}

	if len(pairs) == 0 {
		if len(tidPairs) > 0 {
			return tidPairs, nil
		}
		return nil, fmt.Errorf("TID pairs empty")
	}

	tidPairs = pairs
	tidLastFetch = time.Now()
	return pairs, nil
}

// generateTID generates an x-client-transaction-id for the given API path.
func generateTID(path string) (string, error) {
	pairs, err := fetchTIDPairs()
	if err != nil {
		return "", err
	}

	pair := pairs[rand.IntN(len(pairs))]

	timeNow := int(time.Now().Unix()) - tidEpochOffset
	timeNowBytes := []byte{
		byte(timeNow & 0xff),
		byte((timeNow >> 8) & 0xff),
		byte((timeNow >> 16) & 0xff),
		byte((timeNow >> 24) & 0xff),
	}

	data := fmt.Sprintf("GET!%s!%d%s%s", path, timeNow, tidKeyword, pair.AnimationKey)
	hashBytes := sha256.Sum256([]byte(data))

	keyBytes, err := base64.StdEncoding.DecodeString(pair.Verification)
	if err != nil {
		// Try with padding added
		padded := pair.Verification
		if m := len(padded) % 4; m != 0 {
			padded += strings.Repeat("=", 4-m)
		}
		keyBytes, err = base64.StdEncoding.DecodeString(padded)
		if err != nil {
			return "", fmt.Errorf("decode verification key: %w", err)
		}
	}

	// Build: keyBytes + timeNowBytes + hash[0:16] + [3]
	bytesArr := make([]byte, 0, len(keyBytes)+4+16+1)
	bytesArr = append(bytesArr, keyBytes...)
	bytesArr = append(bytesArr, timeNowBytes...)
	bytesArr = append(bytesArr, hashBytes[:16]...)
	bytesArr = append(bytesArr, 3)

	// XOR with random byte
	randomNum := byte(rand.IntN(256))
	tid := make([]byte, 1+len(bytesArr))
	tid[0] = randomNum
	for i, b := range bytesArr {
		tid[i+1] = b ^ randomNum
	}

	// Base64 without padding
	return strings.TrimRight(base64.StdEncoding.EncodeToString(tid), "="), nil
}
