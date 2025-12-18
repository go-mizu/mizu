package cli

import (
	"testing"
)

func TestMiddlewareDataValidity(t *testing.T) {
	// Ensure we have middlewares
	mws := getMiddlewares()
	if len(mws) == 0 {
		t.Error("no middlewares found")
	}

	// Validate each middleware has required fields
	for _, mw := range mws {
		if mw.Name == "" {
			t.Error("middleware has empty name")
		}
		if mw.Description == "" {
			t.Errorf("middleware %s has empty description", mw.Name)
		}
		if mw.Category == "" {
			t.Errorf("middleware %s has empty category", mw.Name)
		}
		if mw.Import == "" {
			t.Errorf("middleware %s has empty import", mw.Name)
		}
		if mw.QuickStart == "" {
			t.Errorf("middleware %s has empty quick start", mw.Name)
		}
	}
}

func TestFindMiddleware(t *testing.T) {
	// Test finding existing middleware
	mw := findMiddleware("helmet")
	if mw == nil {
		t.Fatal("expected to find helmet middleware")
	}
	if mw.Name != "helmet" {
		t.Errorf("expected name helmet, got %s", mw.Name)
	}

	// Test finding non-existent middleware
	mw = findMiddleware("nonexistent")
	if mw != nil {
		t.Error("expected nil for non-existent middleware")
	}
}

func TestFilterByCategory(t *testing.T) {
	mws := getMiddlewares()
	security := filterByCategory(mws, "security")

	if len(security) == 0 {
		t.Error("expected security middlewares")
	}

	for _, mw := range security {
		if mw.Category != "security" {
			t.Errorf("middleware %s has wrong category %s", mw.Name, mw.Category)
		}
	}
}

func TestGroupByCategory(t *testing.T) {
	mws := getMiddlewares()
	grouped := groupByCategory(mws)

	if len(grouped) == 0 {
		t.Error("expected grouped middlewares")
	}

	// Every middleware should be in exactly one group
	total := 0
	for _, catMws := range grouped {
		total += len(catMws)
	}

	if total != len(mws) {
		t.Errorf("expected %d middlewares, got %d in groups", len(mws), total)
	}
}

func TestAllCategoriesHaveDescription(t *testing.T) {
	for _, cat := range categories {
		if _, ok := categoryDescriptions[cat]; !ok {
			t.Errorf("category %s has no description", cat)
		}
	}
}
