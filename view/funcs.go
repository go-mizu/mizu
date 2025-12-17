package view

import (
	"fmt"
	"html/template"
	"reflect"
	"strings"
)

// baseFuncs returns the base template functions.
// The slot, stack, push, component, partial, and children functions
// are placeholders that get replaced with context-aware versions at render time.
func baseFuncs() template.FuncMap {
	return template.FuncMap{
		// Placeholder functions (replaced at render time)
		"slot":      func(name string, defaults ...any) template.HTML { return "" },
		"stack":     func(name string) template.HTML { return "" },
		"push":      func(name string) string { return "" },
		"component": func(name string, data ...any) template.HTML { return "" },
		"partial":   func(name string, data ...any) template.HTML { return "" },
		"children":  func() template.HTML { return "" },

		// Data helpers
		"dict":    dictFunc,
		"list":    listFunc,
		"default": defaultFunc,
		"empty":   emptyFunc,

		// Safe content
		"safeHTML": safeHTMLFunc,
		"safeCSS":  safeCSSFunc,
		"safeJS":   safeJSFunc,
		"safeURL":  safeURLFunc,

		// String helpers
		"upper":    strings.ToUpper,
		"lower":    strings.ToLower,
		"title":    strings.Title, //nolint:staticcheck // Title is deprecated but still works
		"trim":     strings.TrimSpace,
		"contains": strings.Contains,
		"replace":  strings.ReplaceAll,
		"split":    strings.Split,
		"join":     strings.Join,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,

		// Conditionals
		"ternary":  ternaryFunc,
		"coalesce": coalesceFunc,

		// Comparisons
		"eq": eqFunc,
		"ne": neFunc,
		"lt": ltFunc,
		"le": leFunc,
		"gt": gtFunc,
		"ge": geFunc,

		// Math (basic)
		"add": addFunc,
		"sub": subFunc,
		"mul": mulFunc,
		"div": divFunc,
		"mod": modFunc,
	}
}

// dictFunc creates a map from key-value pairs.
func dictFunc(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("dict requires even number of arguments")
	}
	m := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings, got %T", pairs[i])
		}
		m[key] = pairs[i+1]
	}
	return m, nil
}

// listFunc creates a slice from arguments.
func listFunc(items ...any) []any {
	return items
}

// defaultFunc returns the default value if val is empty.
func defaultFunc(defaultVal, val any) any {
	if emptyFunc(val) {
		return defaultVal
	}
	return val
}

// emptyFunc returns true if val is empty.
func emptyFunc(val any) bool {
	if val == nil {
		return true
	}
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	}
	return false
}

// safeHTMLFunc marks a string as safe HTML.
func safeHTMLFunc(s string) template.HTML {
	return template.HTML(s) //nolint:gosec // intentional
}

// safeCSSFunc marks a string as safe CSS.
func safeCSSFunc(s string) template.CSS {
	return template.CSS(s)
}

// safeJSFunc marks a string as safe JS.
func safeJSFunc(s string) template.JS {
	return template.JS(s) //nolint:gosec // intentional
}

// safeURLFunc marks a string as a safe URL.
func safeURLFunc(s string) template.URL {
	return template.URL(s)
}

// ternaryFunc returns trueVal if cond is true, else falseVal.
func ternaryFunc(cond bool, trueVal, falseVal any) any {
	if cond {
		return trueVal
	}
	return falseVal
}

// coalesceFunc returns the first non-empty value.
func coalesceFunc(vals ...any) any {
	for _, v := range vals {
		if !emptyFunc(v) {
			return v
		}
	}
	return nil
}

// eqFunc returns true if a == b.
func eqFunc(a, b any) bool {
	return reflect.DeepEqual(a, b)
}

// neFunc returns true if a != b.
func neFunc(a, b any) bool {
	return !reflect.DeepEqual(a, b)
}

// ltFunc returns true if a < b (numbers only).
func ltFunc(a, b any) bool {
	av, bv := toFloat64(a), toFloat64(b)
	return av < bv
}

// leFunc returns true if a <= b (numbers only).
func leFunc(a, b any) bool {
	av, bv := toFloat64(a), toFloat64(b)
	return av <= bv
}

// gtFunc returns true if a > b (numbers only).
func gtFunc(a, b any) bool {
	av, bv := toFloat64(a), toFloat64(b)
	return av > bv
}

// geFunc returns true if a >= b (numbers only).
func geFunc(a, b any) bool {
	av, bv := toFloat64(a), toFloat64(b)
	return av >= bv
}

// addFunc returns a + b.
func addFunc(a, b any) any {
	return toFloat64(a) + toFloat64(b)
}

// subFunc returns a - b.
func subFunc(a, b any) any {
	return toFloat64(a) - toFloat64(b)
}

// mulFunc returns a * b.
func mulFunc(a, b any) any {
	return toFloat64(a) * toFloat64(b)
}

// divFunc returns a / b.
func divFunc(a, b any) any {
	bv := toFloat64(b)
	if bv == 0 {
		return 0.0
	}
	return toFloat64(a) / bv
}

// modFunc returns a % b (integers only).
func modFunc(a, b any) any {
	av := toInt64(a)
	bv := toInt64(b)
	if bv == 0 {
		return int64(0)
	}
	return av % bv
}

// toFloat64 converts a value to float64.
func toFloat64(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	}
	return 0
}

// toInt64 converts a value to int64.
func toInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case uint:
		return int64(n)
	case uint8:
		return int64(n)
	case uint16:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		return int64(n)
	case float32:
		return int64(n)
	case float64:
		return int64(n)
	}
	return 0
}
