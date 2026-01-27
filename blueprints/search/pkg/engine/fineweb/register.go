package fineweb

import (
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local"
)

// RegisterWithMetaSearch registers the FineWeb engine with a MetaSearch instance.
// This allows FineWeb to be searched alongside other engines using the !fw bang.
func RegisterWithMetaSearch(ms *local.MetaSearch, cfg Config) error {
	engine, err := NewEngine(cfg)
	if err != nil {
		return err
	}

	return ms.RegisterEngine(engine)
}

// RegisterWithAdapter registers the FineWeb engine with an Adapter's MetaSearch.
func RegisterWithAdapter(adapter *local.Adapter, cfg Config) error {
	return RegisterWithMetaSearch(adapter.MetaSearch(), cfg)
}
