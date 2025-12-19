package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Codec decodes an api document into a Go value.
// contract is stdlib-only, so YAML support must be provided by another package.
type Codec interface {
	Decode(r io.Reader, v any) error
}

// JSONCodec decodes api.json (and can also decode JSON-formatted api.yaml, if desired).
// It is strict by default (unknown fields are rejected).
type JSONCodec struct {
	// Strict rejects unknown fields when true.
	// Default: true.
	Strict bool
}

func (c JSONCodec) Decode(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	if c.Strict || c.Strict == false && c.Strict == (JSONCodec{}).Strict {
		dec.DisallowUnknownFields()
	}
	return dec.Decode(v)
}

// Parse decodes a document using the provided codec and validates it.
func Parse(r io.Reader, c Codec) (*Service, error) {
	if c == nil {
		return nil, fmt.Errorf("contract: nil codec")
	}
	var s Service
	if err := c.Decode(r, &s); err != nil {
		return nil, err
	}
	if err := Validate(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func ParseBytes(b []byte, c Codec) (*Service, error) {
	return Parse(bytes.NewReader(b), c)
}

func ParseString(s string, c Codec) (*Service, error) {
	return Parse(strings.NewReader(s), c)
}

func ParseFile(path string, c Codec) (*Service, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f, c)
}

// UnmarshalJSON allows either `defaults` or `client` in the input JSON.
// If both are present, `defaults` wins.
func (s *Service) UnmarshalJSON(b []byte) error {
	type aux struct {
		Name        string      `json:"name"`
		Description string      `json:"description,omitempty"`
		Defaults    *Defaults   `json:"defaults,omitempty"`
		Client      *Defaults   `json:"client,omitempty"`
		Resources   []*Resource `json:"resources"`
		Types       []*Type     `json:"types,omitempty"`
	}

	var a aux
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&a); err != nil {
		return err
	}

	s.Name = a.Name
	s.Description = a.Description
	s.Resources = a.Resources
	s.Types = a.Types

	if a.Defaults != nil {
		s.Defaults = a.Defaults
	} else {
		s.Defaults = a.Client
	}
	return nil
}
