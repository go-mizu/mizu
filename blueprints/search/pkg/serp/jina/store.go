package jina

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Key represents a stored Jina API key.
type Key struct {
	APIKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
	Balance   int64     `json:"balance"` // last known token balance
	CheckedAt time.Time `json:"checked_at,omitempty"`
}

// KeyStore manages Jina API keys on disk.
type KeyStore struct {
	Keys []Key  `json:"keys"`
	path string
}

// DefaultKeyStorePath returns $HOME/data/jina/keys.json.
func DefaultKeyStorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "jina", "keys.json")
}

// LoadKeyStore loads keys from disk.
func LoadKeyStore(path string) (*KeyStore, error) {
	s := &KeyStore{path: path}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	return s, json.Unmarshal(data, s)
}

// Save writes keys to disk.
func (s *KeyStore) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

// Add adds a new key if not already present.
func (s *KeyStore) Add(apiKey string) {
	for _, k := range s.Keys {
		if k.APIKey == apiKey {
			return
		}
	}
	s.Keys = append(s.Keys, Key{
		APIKey:    apiKey,
		CreatedAt: time.Now(),
		Balance:   10_000_000, // default 10M trial
	})
}

// Remove removes a key by value.
func (s *KeyStore) Remove(apiKey string) {
	kept := s.Keys[:0]
	for _, k := range s.Keys {
		if k.APIKey != apiKey {
			kept = append(kept, k)
		}
	}
	s.Keys = kept
}

// Available returns keys with positive balance.
func (s *KeyStore) Available() []Key {
	var out []Key
	for _, k := range s.Keys {
		if k.Balance > 0 {
			out = append(out, k)
		}
	}
	return out
}

// UpdateBalance updates a key's balance and check time.
func (s *KeyStore) UpdateBalance(apiKey string, balance int64) {
	for i := range s.Keys {
		if s.Keys[i].APIKey == apiKey {
			s.Keys[i].Balance = balance
			s.Keys[i].CheckedAt = time.Now()
			return
		}
	}
}
