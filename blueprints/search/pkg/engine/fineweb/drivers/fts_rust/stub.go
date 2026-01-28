// Package fts_rust is temporarily disabled.
// The actual driver is in driver.go.disabled
package fts_rust

import (
	"context"
	"errors"
	"iter"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

// ErrDisabled is returned when the driver is disabled.
var ErrDisabled = errors.New("fts_rust driver is disabled")

// New returns an error because the driver is disabled.
func New(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
	return nil, ErrDisabled
}

// ListProfiles returns empty list because driver is disabled.
func ListProfiles() []string {
	return nil
}

// StubDriver is a placeholder driver.
type StubDriver struct{}

func (d *StubDriver) Name() string { return "fts_rust" }
func (d *StubDriver) Info() *fineweb.DriverInfo { return &fineweb.DriverInfo{Name: "fts_rust"} }
func (d *StubDriver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	return nil, ErrDisabled
}
func (d *StubDriver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	return ErrDisabled
}
func (d *StubDriver) Count(ctx context.Context) (int64, error) { return 0, nil }
func (d *StubDriver) Close() error { return nil }
