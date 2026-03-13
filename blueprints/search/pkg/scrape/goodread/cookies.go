package goodread

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DefaultCookiesPath is where goodread-tool exports cookies by default.
var DefaultCookiesPath = filepath.Join(os.Getenv("HOME"), "data", "goodread", "cookies.json")

// cookieEntry matches the Playwright cookie dict format written by goodread-tool.
type cookieEntry struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires,omitempty"`
	Secure   bool    `json:"secure,omitempty"`
	HttpOnly bool    `json:"httpOnly,omitempty"`
}

// LoadCookiesFromFile reads a Playwright-format cookies.json and returns
// []*http.Cookie ready for use with NewClientWithCookies.
//
// path defaults to DefaultCookiesPath when empty.
func LoadCookiesFromFile(path string) ([]*http.Cookie, error) {
	if path == "" {
		path = DefaultCookiesPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cookies file %s: %w", path, err)
	}
	var entries []cookieEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse cookies JSON: %w", err)
	}
	cookies := make([]*http.Cookie, 0, len(entries))
	for _, e := range entries {
		if e.Name == "" {
			continue
		}
		c := &http.Cookie{
			Name:     e.Name,
			Value:    e.Value,
			Domain:   e.Domain,
			Path:     e.Path,
			Secure:   e.Secure,
			HttpOnly: e.HttpOnly,
		}
		if e.Expires > 0 {
			c.Expires = time.Unix(int64(e.Expires), 0)
		}
		cookies = append(cookies, c)
	}
	return cookies, nil
}
