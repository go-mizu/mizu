package openlibrarydump

import (
	"compress/gzip"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// importHelper creates temp dumps, imports, and returns the db path.
func importHelper(t *testing.T, authors, works, editions []string, opts Options) string {
	t.Helper()
	tmp := t.TempDir()
	authorsPath := filepath.Join(tmp, "authors.txt.gz")
	worksPath := filepath.Join(tmp, "works.txt.gz")
	editionsPath := filepath.Join(tmp, "editions.txt.gz")
	dbPath := filepath.Join(tmp, "book.duckdb")

	writeGzipLines(t, authorsPath, authors)
	writeGzipLines(t, worksPath, works)
	writeGzipLines(t, editionsPath, editions)

	opts.AuthorsPath = authorsPath
	opts.WorksPath = worksPath
	opts.EditionsPath = editionsPath
	if _, err := ImportToDuckDB(context.Background(), dbPath, opts); err != nil {
		t.Fatalf("import to duckdb: %v", err)
	}
	return dbPath
}

// queryString scans a single string column.
func queryString(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	var s string
	if err := db.QueryRowContext(context.Background(), query, args...).Scan(&s); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return s
}

// queryInt scans a single integer column.
func queryInt(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return n
}

// queryFloat scans a single float column.
func queryFloat(t *testing.T, db *sql.DB, query string, args ...any) float64 {
	t.Helper()
	var f float64
	if err := db.QueryRowContext(context.Background(), query, args...).Scan(&f); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return f
}

func TestExportParquetAndCleanup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmp := t.TempDir()

	authorsPath := filepath.Join(tmp, "authors.txt.gz")
	worksPath := filepath.Join(tmp, "works.txt.gz")
	editionsPath := filepath.Join(tmp, "editions.txt.gz")
	dbPath := filepath.Join(tmp, "book.duckdb")
	parquetDir := filepath.Join(tmp, "parquet")

	writeGzipLines(t, authorsPath, []string{
		"/type/author\t/authors/OL23919A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Jane Austen\",\"bio\":\"English novelist\",\"birth_date\":\"1775\",\"death_date\":\"1817\",\"work_count\":120}",
	})
	writeGzipLines(t, worksPath, []string{
		"/type/work\t/works/OL14986754W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"Pride and Prejudice\",\"description\":\"A classic novel\",\"subjects\":[\"Fiction\",\"Classic\"],\"covers\":[8231856],\"first_publish_date\":\"1813\",\"authors\":[{\"author\":{\"key\":\"/authors/OL23919A\"}}],\"ratings_average\":4.28,\"ratings_count\":3500}",
	})
	writeGzipLines(t, editionsPath, []string{
		"/type/edition\t/books/OL1M\t1\t2020-01-01T00:00:00.000000\t{\"works\":[{\"key\":\"/works/OL14986754W\"}],\"isbn_13\":[\"9780141439518\"],\"isbn_10\":[\"0141439513\"],\"publishers\":[\"Penguin\"],\"publish_date\":\"2002\",\"number_of_pages\":279,\"languages\":[{\"key\":\"/languages/eng\"}]}",
	})

	if _, err := ImportToDuckDB(ctx, dbPath, Options{
		AuthorsPath:  authorsPath,
		WorksPath:    worksPath,
		EditionsPath: editionsPath,
		ReplaceBooks: true,
	}); err != nil {
		t.Fatalf("import to duckdb: %v", err)
	}

	paths, exportStats, err := ExportParquet(ctx, dbPath, parquetDir)
	if err != nil {
		t.Fatalf("export parquet: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("unexpected parquet paths count: %d", len(paths))
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat parquet file %s: %v", p, err)
		}
		if info.Size() <= 0 {
			t.Fatalf("empty parquet file: %s", p)
		}
	}

	if exportStats.BooksExported != 1 {
		t.Fatalf("expected 1 book exported, got %d", exportStats.BooksExported)
	}
	if exportStats.AuthorsExported != 1 {
		t.Fatalf("expected 1 author exported, got %d", exportStats.AuthorsExported)
	}

	// Verify parquet data via DuckDB read_parquet.
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	count := queryInt(t, db, "SELECT COUNT(*) FROM read_parquet(?)", paths[0])
	if count != 1 {
		t.Fatalf("expected 1 book in parquet, got %d", count)
	}

	// Verify rating fields are present in parquet.
	avgRating := queryFloat(t, db, "SELECT average_rating FROM read_parquet(?)", paths[0])
	if avgRating < 4.0 || avgRating > 5.0 {
		t.Fatalf("expected average_rating ~4.28, got %f", avgRating)
	}
	ratingsCount := queryInt(t, db, "SELECT ratings_count FROM read_parquet(?)", paths[0])
	if ratingsCount != 3500 {
		t.Fatalf("expected ratings_count 3500, got %d", ratingsCount)
	}

	if err := DeleteSourceFiles(authorsPath, worksPath, editionsPath); err != nil {
		t.Fatalf("delete source files: %v", err)
	}
	if _, err := os.Stat(authorsPath); !os.IsNotExist(err) {
		t.Fatalf("authors source file still exists")
	}
}

func TestVerifyParquet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmp := t.TempDir()

	authorsPath := filepath.Join(tmp, "authors.txt.gz")
	worksPath := filepath.Join(tmp, "works.txt.gz")
	editionsPath := filepath.Join(tmp, "editions.txt.gz")
	dbPath := filepath.Join(tmp, "book.duckdb")
	parquetDir := filepath.Join(tmp, "parquet")

	writeGzipLines(t, authorsPath, []string{
		"/type/author\t/authors/OL1A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Jane Austen\",\"bio\":\"English novelist\"}",
		"/type/author\t/authors/OL2A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"No Bio Author\"}",
	})
	writeGzipLines(t, worksPath, []string{
		"/type/work\t/works/OL1W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"Pride and Prejudice\",\"subjects\":[\"Fiction\"],\"covers\":[8231856],\"authors\":[{\"author\":{\"key\":\"/authors/OL1A\"}}],\"ratings_average\":4.28,\"ratings_count\":3500}",
		"/type/work\t/works/OL2W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"Unrated Book\",\"subjects\":[],\"authors\":[{\"author\":{\"key\":\"/authors/OL2A\"}}]}",
	})
	writeGzipLines(t, editionsPath, []string{
		"/type/edition\t/books/OL1M\t1\t2020-01-01T00:00:00.000000\t{\"works\":[{\"key\":\"/works/OL1W\"}],\"isbn_13\":[\"9780141439518\"]}",
	})

	if _, err := ImportToDuckDB(ctx, dbPath, Options{
		AuthorsPath:  authorsPath,
		WorksPath:    worksPath,
		EditionsPath: editionsPath,
		ReplaceBooks: true,
	}); err != nil {
		t.Fatalf("import: %v", err)
	}

	paths, _, err := ExportParquet(ctx, dbPath, parquetDir)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	vs, err := VerifyParquet(ctx, paths[0], paths[1])
	if err != nil {
		t.Fatalf("verify: %v", err)
	}

	if vs.BookRows != 2 {
		t.Fatalf("BookRows = %d, want 2", vs.BookRows)
	}
	if vs.WithTitle != 2 {
		t.Fatalf("WithTitle = %d, want 2", vs.WithTitle)
	}
	if vs.WithISBN != 1 {
		t.Fatalf("WithISBN = %d, want 1", vs.WithISBN)
	}
	if vs.WithCover != 1 {
		t.Fatalf("WithCover = %d, want 1", vs.WithCover)
	}
	if vs.WithRating != 1 {
		t.Fatalf("WithRating = %d, want 1", vs.WithRating)
	}
	if vs.AvgRating < 4.0 || vs.AvgRating > 5.0 {
		t.Fatalf("AvgRating = %f, want ~4.28", vs.AvgRating)
	}
	if vs.BookFileSize <= 0 {
		t.Fatalf("BookFileSize = %d, want > 0", vs.BookFileSize)
	}

	if vs.AuthorRows != 2 {
		t.Fatalf("AuthorRows = %d, want 2", vs.AuthorRows)
	}
	if vs.WithName != 2 {
		t.Fatalf("WithName = %d, want 2", vs.WithName)
	}
	if vs.WithBio != 1 {
		t.Fatalf("WithBio = %d, want 1 (only Jane Austen has bio)", vs.WithBio)
	}
	if vs.AuthorFileSize <= 0 {
		t.Fatalf("AuthorFileSize = %d, want > 0", vs.AuthorFileSize)
	}
}

func TestImportMultipleWorksSameAuthor(t *testing.T) {
	t.Parallel()
	dbPath := importHelper(t,
		[]string{
			"/type/author\t/authors/OL23919A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Jane Austen\",\"bio\":\"English novelist\",\"birth_date\":\"1775\",\"death_date\":\"1817\",\"work_count\":120}",
		},
		[]string{
			"/type/work\t/works/OL1W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"Pride and Prejudice\",\"description\":\"A classic\",\"subjects\":[\"Fiction\"],\"first_publish_date\":\"1813\",\"authors\":[{\"author\":{\"key\":\"/authors/OL23919A\"}}]}",
			"/type/work\t/works/OL2W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"Sense and Sensibility\",\"description\":\"Another classic\",\"subjects\":[\"Fiction\",\"Romance\"],\"first_publish_date\":\"1811\",\"authors\":[{\"author\":{\"key\":\"/authors/OL23919A\"}}]}",
		},
		[]string{},
		Options{ReplaceBooks: true, SkipEditions: true},
	)

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	bookCount := queryInt(t, db, "SELECT COUNT(*) FROM books WHERE ol_key LIKE '/works/%'")
	if bookCount != 2 {
		t.Fatalf("expected 2 books, got %d", bookCount)
	}

	authorCount := queryInt(t, db, "SELECT COUNT(*) FROM authors WHERE ol_key LIKE '/authors/%'")
	if authorCount != 1 {
		t.Fatalf("expected 1 author, got %d", authorCount)
	}

	// Both books should reference Jane Austen.
	for _, olKey := range []string{"/works/OL1W", "/works/OL2W"} {
		name := queryString(t, db, "SELECT author_names FROM books WHERE ol_key = ?", olKey)
		if name != "Jane Austen" {
			t.Fatalf("book %s author_names = %q, want Jane Austen", olKey, name)
		}
	}
}

func TestImportMultipleAuthors(t *testing.T) {
	t.Parallel()
	dbPath := importHelper(t,
		[]string{
			"/type/author\t/authors/OL1A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Terry Pratchett\"}",
			"/type/author\t/authors/OL2A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Neil Gaiman\"}",
		},
		[]string{
			"/type/work\t/works/OL10W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"Good Omens\",\"description\":\"The Nice and Accurate Prophecies\",\"subjects\":[\"Fantasy\",\"Humor\"],\"authors\":[{\"author\":{\"key\":\"/authors/OL1A\"}},{\"author\":{\"key\":\"/authors/OL2A\"}}]}",
		},
		[]string{},
		Options{ReplaceBooks: true, SkipEditions: true},
	)

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	authorNames := queryString(t, db, "SELECT author_names FROM books WHERE ol_key = '/works/OL10W'")
	if authorNames != "Terry Pratchett, Neil Gaiman" {
		t.Fatalf("author_names = %q, want 'Terry Pratchett, Neil Gaiman'", authorNames)
	}
}

func TestImportWithoutEditions(t *testing.T) {
	t.Parallel()
	dbPath := importHelper(t,
		[]string{
			"/type/author\t/authors/OL1A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Test Author\"}",
		},
		[]string{
			"/type/work\t/works/OL99W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"No Edition Book\",\"description\":\"A book with no editions\",\"subjects\":[],\"first_publish_date\":\"2020\",\"authors\":[{\"author\":{\"key\":\"/authors/OL1A\"}}]}",
		},
		// No matching editions — tests NULL LEFT JOIN.
		[]string{
			"/type/edition\t/books/OL999M\t1\t2020-01-01T00:00:00.000000\t{\"works\":[{\"key\":\"/works/OL_DIFFERENT_W\"}],\"isbn_13\":[\"9781111111111\"]}",
		},
		Options{ReplaceBooks: true},
	)

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	isbn13 := queryString(t, db, "SELECT isbn13 FROM books WHERE ol_key = '/works/OL99W'")
	if isbn13 != "" {
		t.Fatalf("expected empty isbn13 for unmatched edition, got %q", isbn13)
	}

	publishYear := queryInt(t, db, "SELECT publish_year FROM books WHERE ol_key = '/works/OL99W'")
	if publishYear != 2020 {
		t.Fatalf("expected publish_year 2020 from work, got %d", publishYear)
	}
}

func TestImportDescriptionObjectFormat(t *testing.T) {
	t.Parallel()
	dbPath := importHelper(t,
		[]string{
			"/type/author\t/authors/OL1A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Author One\"}",
		},
		[]string{
			// Description as {value: string} object (common in OL dumps).
			"/type/work\t/works/OL50W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"Object Description Book\",\"description\":{\"type\":\"/type/text\",\"value\":\"This is the real description\"},\"subjects\":[],\"authors\":[{\"author\":{\"key\":\"/authors/OL1A\"}}]}",
			// Description as plain string.
			"/type/work\t/works/OL51W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"String Description Book\",\"description\":\"Plain string description\",\"subjects\":[],\"authors\":[{\"author\":{\"key\":\"/authors/OL1A\"}}]}",
		},
		[]string{},
		Options{ReplaceBooks: true, SkipEditions: true},
	)

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	desc1 := queryString(t, db, "SELECT description FROM books WHERE ol_key = '/works/OL50W'")
	if desc1 != "This is the real description" {
		t.Fatalf("object description = %q, want 'This is the real description'", desc1)
	}

	desc2 := queryString(t, db, "SELECT description FROM books WHERE ol_key = '/works/OL51W'")
	if desc2 != "Plain string description" {
		t.Fatalf("string description = %q, want 'Plain string description'", desc2)
	}
}

func TestImportUnicodeCharacters(t *testing.T) {
	t.Parallel()
	dbPath := importHelper(t,
		[]string{
			"/type/author\t/authors/OL1A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Gabriel García Márquez\"}",
		},
		[]string{
			"/type/work\t/works/OL60W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"Cien años de soledad\",\"description\":\"Una novela del realismo mágico — con «caracteres» especiales\",\"subjects\":[\"Ficción\",\"Литература\",\"文学\"],\"authors\":[{\"author\":{\"key\":\"/authors/OL1A\"}}]}",
		},
		[]string{},
		Options{ReplaceBooks: true, SkipEditions: true},
	)

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	title := queryString(t, db, "SELECT title FROM books WHERE ol_key = '/works/OL60W'")
	if title != "Cien años de soledad" {
		t.Fatalf("unicode title = %q, want 'Cien años de soledad'", title)
	}

	authorName := queryString(t, db, "SELECT author_names FROM books WHERE ol_key = '/works/OL60W'")
	if authorName != "Gabriel García Márquez" {
		t.Fatalf("unicode author = %q, want 'Gabriel García Márquez'", authorName)
	}

	// Verify author table.
	dbAuthor := queryString(t, db, "SELECT name FROM authors WHERE ol_key = '/authors/OL1A'")
	if dbAuthor != "Gabriel García Márquez" {
		t.Fatalf("unicode author in authors table = %q", dbAuthor)
	}
}

func TestImportRatingFields(t *testing.T) {
	t.Parallel()
	dbPath := importHelper(t,
		[]string{
			"/type/author\t/authors/OL1A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Test Author\"}",
		},
		[]string{
			"/type/work\t/works/OL70W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"Rated Book\",\"description\":\"\",\"subjects\":[],\"ratings_average\":3.95,\"ratings_count\":12345,\"authors\":[{\"author\":{\"key\":\"/authors/OL1A\"}}]}",
			"/type/work\t/works/OL71W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"Unrated Book\",\"description\":\"\",\"subjects\":[],\"authors\":[{\"author\":{\"key\":\"/authors/OL1A\"}}]}",
		},
		[]string{},
		Options{ReplaceBooks: true, SkipEditions: true},
	)

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	avg := queryFloat(t, db, "SELECT average_rating FROM books WHERE ol_key = '/works/OL70W'")
	if avg < 3.9 || avg > 4.0 {
		t.Fatalf("expected average_rating ~3.95, got %f", avg)
	}
	rc := queryInt(t, db, "SELECT ratings_count FROM books WHERE ol_key = '/works/OL70W'")
	if rc != 12345 {
		t.Fatalf("expected ratings_count 12345, got %d", rc)
	}

	// Unrated book should have 0 defaults.
	avgUnrated := queryFloat(t, db, "SELECT average_rating FROM books WHERE ol_key = '/works/OL71W'")
	if avgUnrated != 0 {
		t.Fatalf("expected average_rating 0 for unrated, got %f", avgUnrated)
	}
	rcUnrated := queryInt(t, db, "SELECT ratings_count FROM books WHERE ol_key = '/works/OL71W'")
	if rcUnrated != 0 {
		t.Fatalf("expected ratings_count 0 for unrated, got %d", rcUnrated)
	}
}

func TestImportEditionsLanguageMapping(t *testing.T) {
	t.Parallel()
	dbPath := importHelper(t,
		[]string{
			"/type/author\t/authors/OL1A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Author\"}",
		},
		[]string{
			"/type/work\t/works/OL80W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"French Book\",\"subjects\":[],\"authors\":[{\"author\":{\"key\":\"/authors/OL1A\"}}]}",
			"/type/work\t/works/OL81W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"German Book\",\"subjects\":[],\"authors\":[{\"author\":{\"key\":\"/authors/OL1A\"}}]}",
		},
		[]string{
			"/type/edition\t/books/OL80M\t1\t2020-01-01T00:00:00.000000\t{\"works\":[{\"key\":\"/works/OL80W\"}],\"languages\":[{\"key\":\"/languages/fre\"}]}",
			"/type/edition\t/books/OL81M\t1\t2020-01-01T00:00:00.000000\t{\"works\":[{\"key\":\"/works/OL81W\"}],\"languages\":[{\"key\":\"/languages/ger\"}]}",
		},
		Options{ReplaceBooks: true},
	)

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	lang1 := queryString(t, db, "SELECT language FROM books WHERE ol_key = '/works/OL80W'")
	if lang1 != "fr" {
		t.Fatalf("expected language 'fr' for French, got %q", lang1)
	}
	lang2 := queryString(t, db, "SELECT language FROM books WHERE ol_key = '/works/OL81W'")
	if lang2 != "de" {
		t.Fatalf("expected language 'de' for German, got %q", lang2)
	}
}

func TestImportExportFullPipeline(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tmp := t.TempDir()
	authorsPath := filepath.Join(tmp, "authors.txt.gz")
	worksPath := filepath.Join(tmp, "works.txt.gz")
	editionsPath := filepath.Join(tmp, "editions.txt.gz")
	dbPath := filepath.Join(tmp, "book.duckdb")
	parquetDir := filepath.Join(tmp, "parquet")

	writeGzipLines(t, authorsPath, []string{
		"/type/author\t/authors/OL1A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Author A\",\"bio\":{\"type\":\"/type/text\",\"value\":\"Bio of A\"},\"birth_date\":\"1900\"}",
		"/type/author\t/authors/OL2A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Author B\"}",
		// This author is not referenced by any work — should NOT be imported.
		"/type/author\t/authors/OL3A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Unreferenced Author\"}",
	})
	writeGzipLines(t, worksPath, []string{
		"/type/work\t/works/OL100W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"Solo Work\",\"description\":\"By one author\",\"subjects\":[\"Sci-Fi\"],\"covers\":[12345],\"first_publish_date\":\"1999\",\"ratings_average\":4.5,\"ratings_count\":100,\"authors\":[{\"author\":{\"key\":\"/authors/OL1A\"}}]}",
		"/type/work\t/works/OL101W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"Collab Work\",\"description\":{\"type\":\"/type/text\",\"value\":\"By two authors\"},\"subjects\":[\"Drama\",\"Comedy\"],\"first_publish_date\":\"2005\",\"authors\":[{\"author\":{\"key\":\"/authors/OL1A\"}},{\"author\":{\"key\":\"/authors/OL2A\"}}]}",
		"/type/work\t/works/OL102W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"No Author Work\",\"description\":\"\",\"subjects\":[],\"authors\":[]}",
	})
	writeGzipLines(t, editionsPath, []string{
		"/type/edition\t/books/OL100M\t1\t2020-01-01T00:00:00.000000\t{\"works\":[{\"key\":\"/works/OL100W\"}],\"isbn_13\":[\"978-0-123456-78-9\"],\"isbn_10\":[\"0123456789\"],\"publishers\":[\"Big Pub\"],\"publish_date\":\"January 2000\",\"number_of_pages\":350,\"languages\":[{\"key\":\"/languages/eng\"}]}",
	})

	stats, err := ImportToDuckDB(ctx, dbPath, Options{
		AuthorsPath:  authorsPath,
		WorksPath:    worksPath,
		EditionsPath: editionsPath,
		ReplaceBooks: true,
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if stats.WorksStaged != 3 {
		t.Fatalf("works staged = %d, want 3", stats.WorksStaged)
	}
	if stats.AuthorsStaged != 2 {
		t.Fatalf("authors staged = %d, want 2 (unreferenced excluded)", stats.AuthorsStaged)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	// "No Author Work" should have "Unknown" as author.
	authorName := queryString(t, db, "SELECT author_names FROM books WHERE ol_key = '/works/OL102W'")
	if authorName != "Unknown" {
		t.Fatalf("no-author book author_names = %q, want 'Unknown'", authorName)
	}

	// Solo Work should have cover URL.
	coverURL := queryString(t, db, "SELECT cover_url FROM books WHERE ol_key = '/works/OL100W'")
	if coverURL != "https://covers.openlibrary.org/b/id/12345-M.jpg" {
		t.Fatalf("cover_url = %q", coverURL)
	}

	// ISBN cleaning: "978-0-123456-78-9" → digits only.
	isbn13 := queryString(t, db, "SELECT isbn13 FROM books WHERE ol_key = '/works/OL100W'")
	if isbn13 != "9780123456789" {
		t.Fatalf("isbn13 = %q, want '9780123456789' (cleaned)", isbn13)
	}

	// Collab Work should have both authors.
	collabAuthors := queryString(t, db, "SELECT author_names FROM books WHERE ol_key = '/works/OL101W'")
	if collabAuthors != "Author A, Author B" {
		t.Fatalf("collab author_names = %q", collabAuthors)
	}

	// Unreferenced author should not be in authors table.
	unreferenced := queryInt(t, db, "SELECT COUNT(*) FROM authors WHERE ol_key = '/authors/OL3A'")
	if unreferenced != 0 {
		t.Fatalf("unreferenced author should not be imported, got count=%d", unreferenced)
	}

	// Export and verify parquet.
	paths, exportStats, err := ExportParquet(ctx, dbPath, parquetDir)
	if err != nil {
		t.Fatalf("export parquet: %v", err)
	}
	if exportStats.BooksExported != 3 {
		t.Fatalf("books exported = %d, want 3", exportStats.BooksExported)
	}
	if exportStats.AuthorsExported != 2 {
		t.Fatalf("authors exported = %d, want 2", exportStats.AuthorsExported)
	}

	// Verify parquet books have all key fields.
	pqDB, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open in-memory duckdb: %v", err)
	}
	defer pqDB.Close()

	pqRating := queryFloat(t, pqDB, "SELECT average_rating FROM read_parquet(?) WHERE ol_key = '/works/OL100W'", paths[0])
	if pqRating < 4.4 || pqRating > 4.6 {
		t.Fatalf("parquet average_rating = %f, want ~4.5", pqRating)
	}
	pqISBN := queryString(t, pqDB, "SELECT isbn13 FROM read_parquet(?) WHERE ol_key = '/works/OL100W'", paths[0])
	if pqISBN != "9780123456789" {
		t.Fatalf("parquet isbn13 = %q", pqISBN)
	}
}

func writeGzipLines(t *testing.T, path string, lines []string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()

	zw := gzip.NewWriter(f)
	for _, line := range lines {
		if _, err := zw.Write([]byte(line + "\n")); err != nil {
			t.Fatalf("write gzip %s: %v", path, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close gzip %s: %v", path, err)
	}
}
