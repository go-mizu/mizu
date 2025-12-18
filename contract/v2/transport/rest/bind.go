package rest

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"
)

// fillFromPath fills struct fields from path parameters using mizu.Ctx.Param.
func fillFromPath(dst any, c *mizu.Ctx, params []string) error {
	if len(params) == 0 {
		return nil
	}

	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("dst must be non-nil pointer")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("dst must point to struct")
	}
	t := v.Type()

	for _, param := range params {
		value := c.Param(param)
		if value == "" {
			continue
		}

		fi, ok := findField(t, param)
		if !ok {
			continue
		}

		fv := v.Field(fi)
		if !fv.CanSet() {
			continue
		}

		if err := setFieldFromString(fv, value); err != nil {
			return fmt.Errorf("%s: %w", param, err)
		}
	}
	return nil
}

// fillFromQuery fills struct fields from query parameters.
func fillFromQuery(dst any, values url.Values) error {
	if len(values) == 0 {
		return nil
	}

	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("dst must be non-nil pointer")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("dst must point to struct")
	}
	t := v.Type()

	for key, vs := range values {
		if len(vs) == 0 {
			continue
		}

		fi, ok := findField(t, key)
		if !ok {
			continue
		}

		fv := v.Field(fi)
		if !fv.CanSet() {
			continue
		}

		if err := setFieldFromString(fv, vs[0]); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	return nil
}

// findField finds a struct field by wire name (json tag or field name).
func findField(t reflect.Type, wire string) (int, bool) {
	wire = strings.TrimSpace(wire)
	if wire == "" {
		return 0, false
	}
	wireLower := strings.ToLower(wire)

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}
		if sf.Anonymous {
			continue
		}

		// Check json tag first
		tag := sf.Tag.Get("json")
		if tag != "" {
			name := strings.Split(tag, ",")[0]
			name = strings.TrimSpace(name)
			if name == "-" {
				continue
			}
			if strings.ToLower(name) == wireLower {
				return i, true
			}
		}

		// Fall back to field name
		if strings.ToLower(sf.Name) == wireLower {
			return i, true
		}
	}
	return 0, false
}

// setFieldFromString sets a reflect.Value from a string.
func setFieldFromString(fv reflect.Value, raw string) error {
	// Handle pointer types by allocating
	if fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		return setFieldFromString(fv.Elem(), raw)
	}

	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
		return nil

	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		fv.SetBool(b)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetInt(n)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetUint(n)
		return nil

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetFloat(f)
		return nil
	}

	return fmt.Errorf("unsupported type %s", fv.Type())
}

