package hypermedia

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "hello"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	var data map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Should have _links with self
	links, ok := data["_links"].([]any)
	if !ok {
		t.Fatal("expected _links array")
	}

	if len(links) == 0 {
		t.Error("expected at least one link")
	}
}

func TestSelfLink(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		BaseURL:  "https://api.example.com",
		SelfLink: true,
	}))

	app.Get("/users/123", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"id": "123"})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var data map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &data)

	links := data["_links"].([]any)
	selfLink := links[0].(map[string]any)

	if selfLink["href"] != "https://api.example.com/users/123" {
		t.Errorf("expected self link, got %q", selfLink["href"])
	}
	if selfLink["rel"] != "self" {
		t.Errorf("expected rel=self, got %q", selfLink["rel"])
	}
}

func TestAddLink(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		BaseURL:  "https://api.example.com",
		SelfLink: false,
	}))

	app.Get("/users/123", func(c *mizu.Ctx) error {
		AddLink(c, Link{
			Href: "https://api.example.com/users/123/posts",
			Rel:  "posts",
		})
		return c.JSON(http.StatusOK, map[string]string{"id": "123"})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var data map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &data)

	links := data["_links"].([]any)
	if len(links) != 1 {
		t.Errorf("expected 1 link, got %d", len(links))
	}

	link := links[0].(map[string]any)
	if link["rel"] != "posts" {
		t.Errorf("expected rel=posts, got %q", link["rel"])
	}
}

func TestAddLinks(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{SelfLink: false}))

	app.Get("/users/123", func(c *mizu.Ctx) error {
		AddLinks(c,
			Link{Href: "/users/123/posts", Rel: "posts"},
			Link{Href: "/users/123/comments", Rel: "comments"},
		)
		return c.JSON(http.StatusOK, map[string]string{"id": "123"})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var data map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &data)

	links := data["_links"].([]any)
	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d", len(links))
	}
}

func TestLinkProvider(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SelfLink: false,
		LinkProvider: func(path string, method string) Links {
			if path == "/users" {
				return Links{
					{Href: "/users", Rel: "create", Method: "POST"},
				}
			}
			return nil
		},
	}))

	app.Get("/users", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, []string{"user1", "user2"})
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Response will be an array, links won't be added to arrays
	// This test verifies the middleware doesn't crash on arrays
	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestNonJSONResponse(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/text", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "plain text")
	})

	req := httptest.NewRequest(http.MethodGet, "/text", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "plain text" {
		t.Errorf("expected plain text unchanged, got %q", rec.Body.String())
	}
}

func TestCustomLinksKey(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		LinksKey: "links",
		SelfLink: true,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"data": "test"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var data map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &data)

	if _, ok := data["links"]; !ok {
		t.Error("expected custom links key")
	}
	if _, ok := data["_links"]; ok {
		t.Error("expected no _links key")
	}
}

func TestResource(t *testing.T) {
	resource := NewResource(
		map[string]string{"id": "123", "name": "Test"},
		Link{Href: "/items/123", Rel: "self"},
		Link{Href: "/items/123/related", Rel: "related"},
	)

	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed map[string]any
	_ = json.Unmarshal(data, &parsed)

	if _, ok := parsed["data"]; !ok {
		t.Error("expected data field")
	}
	if _, ok := parsed["_links"]; !ok {
		t.Error("expected _links field")
	}
}

//nolint:cyclop // Test function with multiple link type assertions
func TestCollection(t *testing.T) {
	items := []string{"item1", "item2"}
	collection := NewCollection(items, 100, 2, 10, "/items")

	if collection.Total != 100 {
		t.Errorf("expected total 100, got %d", collection.Total)
	}
	if collection.Page != 2 {
		t.Errorf("expected page 2, got %d", collection.Page)
	}
	if collection.TotalPages != 10 {
		t.Errorf("expected 10 total pages, got %d", collection.TotalPages)
	}

	// Check links
	hasFirst := false
	hasPrev := false
	hasNext := false
	hasLast := false

	for _, link := range collection.Links {
		switch link.Rel {
		case "first":
			hasFirst = true
		case "prev":
			hasPrev = true
		case "next":
			hasNext = true
		case "last":
			hasLast = true
		}
	}

	if !hasFirst {
		t.Error("expected first link")
	}
	if !hasPrev {
		t.Error("expected prev link on page 2")
	}
	if !hasNext {
		t.Error("expected next link")
	}
	if !hasLast {
		t.Error("expected last link")
	}
}

func TestCollectionFirstPage(t *testing.T) {
	collection := NewCollection([]string{}, 50, 1, 10, "/items")

	hasPrev := false
	for _, link := range collection.Links {
		if link.Rel == "prev" {
			hasPrev = true
		}
	}

	if hasPrev {
		t.Error("expected no prev link on first page")
	}
}

func TestCollectionLastPage(t *testing.T) {
	collection := NewCollection([]string{}, 50, 5, 10, "/items")

	hasNext := false
	for _, link := range collection.Links {
		if link.Rel == "next" {
			hasNext = true
		}
	}

	if hasNext {
		t.Error("expected no next link on last page")
	}
}

func TestHAL(t *testing.T) {
	hal := NewHAL(map[string]any{
		"id":   "123",
		"name": "Test User",
	})

	hal.AddLink("self", Link{Href: "/users/123"})
	hal.AddLink("posts", Link{Href: "/users/123/posts"})

	data, err := json.Marshal(hal)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed map[string]any
	_ = json.Unmarshal(data, &parsed)

	if parsed["id"] != "123" {
		t.Error("expected id property")
	}
	if parsed["name"] != "Test User" {
		t.Error("expected name property")
	}

	links, ok := parsed["_links"].(map[string]any)
	if !ok {
		t.Fatal("expected _links object")
	}

	if _, ok := links["self"]; !ok {
		t.Error("expected self link")
	}
}

func TestHALEmbedded(t *testing.T) {
	user := NewHAL(map[string]any{"id": "123", "name": "User"})

	post1 := *NewHAL(map[string]any{"id": "1", "title": "Post 1"})
	post2 := *NewHAL(map[string]any{"id": "2", "title": "Post 2"})

	user.Embed("posts", post1, post2)

	data, _ := json.Marshal(user)

	var parsed map[string]any
	_ = json.Unmarshal(data, &parsed)

	embedded, ok := parsed["_embedded"].(map[string]any)
	if !ok {
		t.Fatal("expected _embedded object")
	}

	posts, ok := embedded["posts"].([]any)
	if !ok {
		t.Fatal("expected posts array")
	}

	if len(posts) != 2 {
		t.Errorf("expected 2 embedded posts, got %d", len(posts))
	}
}

func TestGetLinks(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{SelfLink: false}))

	var gotLinks Links

	app.Get("/", func(c *mizu.Ctx) error {
		AddLink(c, Link{Href: "/test", Rel: "test"})
		gotLinks = GetLinks(c)
		return c.JSON(http.StatusOK, map[string]string{})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if len(gotLinks) != 1 {
		t.Errorf("expected 1 link from GetLinks, got %d", len(gotLinks))
	}
}
