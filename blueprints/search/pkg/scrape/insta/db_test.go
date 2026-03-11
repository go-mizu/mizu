package insta

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDB_InsertAndQuery(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.duckdb")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	// Insert posts
	posts := []Post{
		{
			ID: "1", Shortcode: "AAA", TypeName: "GraphImage",
			DisplayURL: "https://example.com/1.jpg",
			LikeCount: 1000, CommentCount: 50,
			TakenAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			OwnerID: "u1", OwnerName: "user1",
			FetchedAt: time.Now(),
		},
		{
			ID: "2", Shortcode: "BBB", TypeName: "GraphVideo",
			DisplayURL: "https://example.com/2.jpg",
			VideoURL: "https://example.com/2.mp4",
			IsVideo: true, LikeCount: 500, ViewCount: 10000,
			TakenAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			OwnerID: "u1", OwnerName: "user1",
			FetchedAt: time.Now(),
		},
		{
			ID: "3", Shortcode: "CCC", TypeName: "GraphSidecar",
			DisplayURL: "https://example.com/3.jpg",
			LikeCount: 2000,
			TakenAt: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			OwnerID: "u1", OwnerName: "user1",
			Children: []Post{{ID: "c1"}, {ID: "c2"}},
			FetchedAt: time.Now(),
		},
	}

	if err := db.InsertPosts(posts); err != nil {
		t.Fatalf("InsertPosts: %v", err)
	}

	// Get stats
	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.Posts != 3 {
		t.Errorf("Posts = %d, want 3", stats.Posts)
	}
	if stats.DBSize <= 0 {
		t.Errorf("DBSize = %d, want > 0", stats.DBSize)
	}

	// Top posts
	top, err := db.TopPosts(2)
	if err != nil {
		t.Fatalf("TopPosts: %v", err)
	}
	if len(top) != 2 {
		t.Fatalf("top = %d, want 2", len(top))
	}
	// Post 3 (2000 likes) should be first, then Post 1 (1000 likes)
	if top[0].Shortcode != "CCC" {
		t.Errorf("top[0].Shortcode = %q, want CCC", top[0].Shortcode)
	}
	if top[1].Shortcode != "AAA" {
		t.Errorf("top[1].Shortcode = %q, want AAA", top[1].Shortcode)
	}
}

func TestDB_InsertComments(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.duckdb")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	comments := []Comment{
		{ID: "c1", PostID: "p1", Text: "Great post!", AuthorID: "a1", AuthorName: "commenter1", LikeCount: 5, CreatedAt: time.Now()},
		{ID: "c2", PostID: "p1", Text: "Amazing!", AuthorID: "a2", AuthorName: "commenter2", LikeCount: 2, CreatedAt: time.Now()},
	}

	if err := db.InsertComments(comments); err != nil {
		t.Fatalf("InsertComments: %v", err)
	}

	stats, _ := db.GetStats()
	if stats.Comments != 2 {
		t.Errorf("Comments = %d, want 2", stats.Comments)
	}
}

func TestDB_InsertMedia(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.duckdb")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	items := []MediaItem{
		{URL: "https://example.com/1.jpg", PostID: "p1", Shortcode: "AAA", Type: "image", Width: 1080, Height: 1080},
		{URL: "https://example.com/2.mp4", PostID: "p1", Shortcode: "AAA", Type: "video", Width: 1080, Height: 1920},
		{URL: "https://example.com/3.jpg", PostID: "p2", Shortcode: "BBB", Type: "image", Width: 1080, Height: 1080, Index: 1},
	}

	if err := db.InsertMedia(items); err != nil {
		t.Fatalf("InsertMedia: %v", err)
	}

	stats, _ := db.GetStats()
	if stats.Media != 3 {
		t.Errorf("Media = %d, want 3", stats.Media)
	}
}

func TestDB_Upsert(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.duckdb")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	// Insert, then insert again with updated data
	posts := []Post{
		{ID: "1", Shortcode: "AAA", TypeName: "GraphImage", DisplayURL: "https://example.com/1.jpg", LikeCount: 100, FetchedAt: time.Now()},
	}
	if err := db.InsertPosts(posts); err != nil {
		t.Fatalf("InsertPosts 1: %v", err)
	}

	posts[0].LikeCount = 200
	if err := db.InsertPosts(posts); err != nil {
		t.Fatalf("InsertPosts 2: %v", err)
	}

	stats, _ := db.GetStats()
	if stats.Posts != 1 {
		t.Errorf("Posts = %d, want 1 (upsert)", stats.Posts)
	}

	top, _ := db.TopPosts(1)
	if len(top) != 1 || top[0].LikeCount != 200 {
		t.Errorf("LikeCount = %d, want 200 (updated)", top[0].LikeCount)
	}
}

func TestDB_EmptyInsert(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.duckdb")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	// Empty slices should be no-ops
	if err := db.InsertPosts(nil); err != nil {
		t.Errorf("InsertPosts(nil): %v", err)
	}
	if err := db.InsertComments(nil); err != nil {
		t.Errorf("InsertComments(nil): %v", err)
	}
	if err := db.InsertMedia(nil); err != nil {
		t.Errorf("InsertMedia(nil): %v", err)
	}
}

func TestDB_Path(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.duckdb")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	if db.Path() != dbPath {
		t.Errorf("Path = %q, want %q", db.Path(), dbPath)
	}
}

func TestDB_CreatesDir(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "subdir", "nested", "test.duckdb")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	db.Close()

	if _, err := os.Stat(filepath.Dir(dbPath)); os.IsNotExist(err) {
		t.Error("DB directory was not created")
	}
}
