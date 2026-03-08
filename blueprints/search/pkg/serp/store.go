package serp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Account struct {
	Email        string    `json:"email"`
	Password     string    `json:"password,omitempty"`
	APIKey       string    `json:"api_key"`
	Provider     string    `json:"provider,omitempty"` // serper, zenserp, searchapi, serpstack, serply, serpapi
	RegisteredAt time.Time `json:"registered_at"`
	SearchesLeft int       `json:"searches_left"`
	LastChecked  time.Time `json:"last_checked"`
}

type Store struct {
	Accounts []Account `json:"accounts"`
	path     string
}

func DefaultStorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "serp", "keys.json")
}

func LoadStore(path string) (*Store, error) {
	s := &Store{path: path}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	return s, json.Unmarshal(data, s)
}

func (s *Store) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *Store) Add(a Account) {
	s.Accounts = append(s.Accounts, a)
}

// Available returns accounts with searches_left >= 10.
func (s *Store) Available() []Account {
	var out []Account
	for _, a := range s.Accounts {
		if a.SearchesLeft >= 10 {
			out = append(out, a)
		}
	}
	return out
}

// PruneExhausted removes accounts with searches_left == 0 (auto-remove only when exactly 0).
func (s *Store) PruneExhausted() (removed int) {
	kept := s.Accounts[:0]
	for _, a := range s.Accounts {
		if a.SearchesLeft == 0 {
			removed++
		} else {
			kept = append(kept, a)
		}
	}
	s.Accounts = kept
	return
}

// ByProvider returns accounts for a specific provider.
func (s *Store) ByProvider(provider string) []Account {
	var out []Account
	for _, a := range s.Accounts {
		if a.Provider == provider {
			out = append(out, a)
		}
	}
	return out
}

// UpdateSearchesLeft finds account by api_key and updates searches_left.
func (s *Store) UpdateSearchesLeft(apiKey string, left int) {
	for i := range s.Accounts {
		if s.Accounts[i].APIKey == apiKey {
			s.Accounts[i].SearchesLeft = left
			s.Accounts[i].LastChecked = time.Now()
			return
		}
	}
}

// RemoveKey removes an account by API key.
func (s *Store) RemoveKey(apiKey string) {
	kept := s.Accounts[:0]
	for _, a := range s.Accounts {
		if a.APIKey != apiKey {
			kept = append(kept, a)
		}
	}
	s.Accounts = kept
}
