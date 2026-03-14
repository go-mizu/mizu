package hn2

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// parseIntAny converts a JSON-decoded value (float64, int64, json.Number, or string)
// to int64. ClickHouse returns numeric types as JSON numbers or quoted strings
// depending on the query format and column type.
func parseIntAny(v any) (int64, error) {
	switch x := v.(type) {
	case float64:
		return int64(x), nil
	case int64:
		return x, nil
	case int:
		return int64(x), nil
	case json.Number:
		return x.Int64()
	case string:
		return strconv.ParseInt(strings.TrimSpace(x), 10, 64)
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", v)
	}
}

// parseFloatAny converts a JSON-decoded value to float64.
func parseFloatAny(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case json.Number:
		f, _ := x.Float64()
		return f
	case string:
		var f float64
		fmt.Sscanf(strings.TrimSpace(x), "%f", &f)
		return f
	default:
		n, _ := parseIntAny(v)
		return float64(n)
	}
}
