//go:build chdb

package duckdb

import (
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/metastore"
)

func init() {
	metastore.Register("duckdb", unavailableDriver{})
}

type unavailableDriver struct{}

func (unavailableDriver) Open(string, metastore.Options) (metastore.Store, error) {
	return nil, fmt.Errorf("duckdb metastore driver unavailable in chdb build")
}
