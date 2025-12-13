package msgpack

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"nil", nil},
		{"true", true},
		{"false", false},
		{"positive int", int64(42)},
		{"negative int", int64(-42)},
		{"zero", int64(0)},
		{"large int", int64(1000000)},
		{"uint", uint64(255)},
		{"float64", float64(3.14)},
		{"string", "hello"},
		{"empty string", ""},
		{"long string", "this is a longer string that exceeds 31 bytes"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := Marshal(tc.value)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			result, err := Unmarshal(data)
			if err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			switch v := tc.value.(type) {
			case nil:
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			case bool:
				if result != v {
					t.Errorf("expected %v, got %v", v, result)
				}
			case int64:
				if result != v {
					t.Errorf("expected %v, got %v", v, result)
				}
			case uint64:
				if result != v {
					t.Errorf("expected %v, got %v", v, result)
				}
			case float64:
				if result != v {
					t.Errorf("expected %v, got %v", v, result)
				}
			case string:
				if result != v {
					t.Errorf("expected %q, got %q", v, result)
				}
			}
		})
	}
}

func TestMarshalArray(t *testing.T) {
	arr := []any{int64(1), int64(2), int64(3)}
	data, err := Marshal(arr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	result, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	resultArr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}

	if len(resultArr) != 3 {
		t.Errorf("expected 3 elements, got %d", len(resultArr))
	}
}

func TestMarshalMap(t *testing.T) {
	m := map[string]any{
		"name": "John",
		"age":  int64(30),
	}

	data, err := Marshal(m)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	result, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if resultMap["name"] != "John" {
		t.Errorf("expected John, got %v", resultMap["name"])
	}
}

func TestMarshalBinary(t *testing.T) {
	bin := []byte{0x01, 0x02, 0x03}
	data, err := Marshal(bin)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	result, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	resultBin, ok := result.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", result)
	}

	if !bytes.Equal(resultBin, bin) {
		t.Errorf("expected %v, got %v", bin, resultBin)
	}
}

func TestResponse(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return Response(c, http.StatusOK, map[string]any{"hello": "world"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Header().Get("Content-Type") != ContentType {
		t.Errorf("expected content type %q, got %q", ContentType, rec.Header().Get("Content-Type"))
	}

	// Decode response
	result, err := Unmarshal(rec.Body.Bytes())
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["hello"] != "world" {
		t.Errorf("expected world, got %v", m["hello"])
	}
}

func TestBind(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Post("/", func(c *mizu.Ctx) error {
		data, err := Bind(c)
		if err != nil {
			return c.Text(http.StatusBadRequest, err.Error())
		}

		m, ok := data.(map[string]any)
		if !ok {
			return c.Text(http.StatusBadRequest, "not a map")
		}

		name, _ := m["name"].(string)
		return c.Text(http.StatusOK, name)
	})

	body, _ := Marshal(map[string]any{"name": "Alice"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", ContentType)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "Alice" {
		t.Errorf("expected Alice, got %q", rec.Body.String())
	}
}

func TestBody(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Post("/", func(c *mizu.Ctx) error {
		body := Body(c)
		if body == nil {
			return c.Text(http.StatusBadRequest, "no body")
		}
		return c.Text(http.StatusOK, "got body")
	})

	body, _ := Marshal("test")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", ContentType)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "got body" {
		t.Errorf("expected got body, got %q", rec.Body.String())
	}
}

func TestEncoderIntegers(t *testing.T) {
	tests := []struct {
		name  string
		value int64
	}{
		{"zero", 0},
		{"positive fixint", 100},
		{"negative fixint", -20},
		{"int8 positive", 127},
		{"int8 negative", -128},
		{"int16 positive", 1000},
		{"int16 negative", -1000},
		{"int32 positive", 100000},
		{"int32 negative", -100000},
		{"int64 positive", 5000000000},
		{"int64 negative", -5000000000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := Marshal(tc.value)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			result, err := Unmarshal(data)
			if err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			if result != tc.value {
				t.Errorf("expected %d, got %v", tc.value, result)
			}
		})
	}
}

func TestEncoderFloats(t *testing.T) {
	t.Run("float32", func(t *testing.T) {
		data, err := Marshal(float32(3.14))
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		result, err := Unmarshal(data)
		if err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		f, ok := result.(float32)
		if !ok {
			t.Fatalf("expected float32, got %T", result)
		}

		if f < 3.13 || f > 3.15 {
			t.Errorf("expected ~3.14, got %v", f)
		}
	})

	t.Run("float64", func(t *testing.T) {
		data, err := Marshal(float64(3.14159265358979))
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		result, err := Unmarshal(data)
		if err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		f, ok := result.(float64)
		if !ok {
			t.Fatalf("expected float64, got %T", result)
		}

		if f < 3.14 || f > 3.15 {
			t.Errorf("expected ~3.14, got %v", f)
		}
	})
}

func TestUnsupportedType(t *testing.T) {
	type custom struct{}
	_, err := Marshal(custom{})
	if !errors.Is(err, ErrUnsupportedType) {
		t.Errorf("expected ErrUnsupportedType, got %v", err)
	}
}

func TestEmptyBuffer(t *testing.T) {
	_, err := Unmarshal([]byte{})
	if err == nil {
		t.Error("expected error for empty buffer")
	}
}
