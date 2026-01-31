package tools

import (
	"context"
	"testing"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()

	tool := &Tool{
		Name:        "search",
		Description: "Search the web",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The search query",
				},
			},
			"required": []string{"query"},
		},
		Execute: func(ctx context.Context, input map[string]any) (string, error) {
			return "results for: " + input["query"].(string), nil
		},
	}

	reg.Register(tool)

	got := reg.Get("search")
	if got == nil {
		t.Fatal("expected tool, got nil")
	}
	if got.Name != "search" {
		t.Errorf("Name = %q, want %q", got.Name, "search")
	}
	if got.Description != "Search the web" {
		t.Errorf("Description = %q, want %q", got.Description, "Search the web")
	}
	if got.InputSchema == nil {
		t.Fatal("InputSchema is nil")
	}
	if got.InputSchema["type"] != "object" {
		t.Errorf("InputSchema[type] = %v, want %q", got.InputSchema["type"], "object")
	}

	// Execute should work.
	result, err := got.Execute(context.Background(), map[string]any{"query": "hello"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result != "results for: hello" {
		t.Errorf("Execute result = %q, want %q", result, "results for: hello")
	}

	// Unknown name returns nil.
	if reg.Get("unknown") != nil {
		t.Error("expected nil for unknown tool")
	}
}

func TestRegistry_All(t *testing.T) {
	reg := NewRegistry()

	tools := []*Tool{
		{Name: "alpha", Description: "Tool A"},
		{Name: "beta", Description: "Tool B"},
		{Name: "gamma", Description: "Tool C"},
	}
	for _, tool := range tools {
		reg.Register(tool)
	}

	all := reg.All()
	if len(all) != 3 {
		t.Fatalf("All() returned %d tools, want 3", len(all))
	}

	// Verify all names are present (order is not guaranteed).
	names := make(map[string]bool)
	for _, tool := range all {
		names[tool.Name] = true
	}
	for _, want := range []string{"alpha", "beta", "gamma"} {
		if !names[want] {
			t.Errorf("All() missing tool %q", want)
		}
	}
}

func TestRegistry_Definitions(t *testing.T) {
	reg := NewRegistry()

	reg.Register(&Tool{
		Name:        "calculator",
		Description: "Perform arithmetic",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"expression": map[string]any{
					"type":        "string",
					"description": "Math expression to evaluate",
				},
			},
			"required": []string{"expression"},
		},
	})
	reg.Register(&Tool{
		Name:        "weather",
		Description: "Get weather forecast",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{
					"type":        "string",
					"description": "City name",
				},
			},
		},
	})

	defs := reg.Definitions()
	if len(defs) != 2 {
		t.Fatalf("Definitions() returned %d entries, want 2", len(defs))
	}

	// Build a lookup by name for order-independent checking.
	byName := make(map[string]map[string]any)
	for _, d := range defs {
		name, ok := d["name"].(string)
		if !ok {
			t.Fatal("definition missing 'name' string key")
		}
		byName[name] = d
	}

	// Verify calculator definition.
	calc, ok := byName["calculator"]
	if !ok {
		t.Fatal("missing definition for 'calculator'")
	}
	if calc["description"] != "Perform arithmetic" {
		t.Errorf("calculator description = %v, want %q", calc["description"], "Perform arithmetic")
	}
	if calc["input_schema"] == nil {
		t.Error("calculator input_schema is nil")
	}

	// Verify weather definition.
	wth, ok := byName["weather"]
	if !ok {
		t.Fatal("missing definition for 'weather'")
	}
	if wth["description"] != "Get weather forecast" {
		t.Errorf("weather description = %v, want %q", wth["description"], "Get weather forecast")
	}
	if wth["input_schema"] == nil {
		t.Error("weather input_schema is nil")
	}

	// Verify each definition has exactly the three expected keys.
	for _, d := range defs {
		for _, key := range []string{"name", "description", "input_schema"} {
			if _, exists := d[key]; !exists {
				t.Errorf("definition %v missing key %q", d["name"], key)
			}
		}
		if len(d) != 3 {
			t.Errorf("definition %v has %d keys, want 3", d["name"], len(d))
		}
	}
}
