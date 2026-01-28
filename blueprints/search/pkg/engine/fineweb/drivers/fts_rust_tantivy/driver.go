// Package fts_rust_tantivy provides fts_rust driver with tantivy profile.
package fts_rust_tantivy

import (
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_rust"
)

func init() {
	fineweb.Register("fts_rust_tantivy", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		// Override profile to tantivy
		if cfg.Options == nil {
			cfg.Options = make(map[string]any)
		}
		cfg.Options["profile"] = "tantivy"
		return fts_rust.New(cfg)
	})
}
