// Package data provides embedded benchmark data files.
package data

import _ "embed"

// QueriesJSONL is the embedded queries.jsonl file with benchmark query entries.
//
//go:embed queries.jsonl
var QueriesJSONL []byte
