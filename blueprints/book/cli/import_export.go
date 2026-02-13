package cli

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/pkg/openlibrarydump"
	"github.com/go-mizu/mizu/blueprints/book/store/factory"
	"github.com/go-mizu/mizu/blueprints/book/types"
	"github.com/spf13/cobra"
)

func NewImportExport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import/export books",
	}
	cmd.AddCommand(newImportCSV())
	cmd.AddCommand(newImportOpenLibrary())
	cmd.AddCommand(newExportCSV())
	return cmd
}

func runOpenLibraryImport(ctx context.Context, opts openlibrarydump.Options, dbPath, parquetDir string, exportParquet, cleanupSource bool) error {
	resolved, err := openlibrarydump.ResolvePaths(opts)
	if err != nil {
		return err
	}
	stats, err := openlibrarydump.ImportToDuckDB(ctx, dbPath, resolved)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  Import complete ✓"))
	fmt.Printf("  %-20s%s\n", "Works staged", formatCount(stats.WorksStaged))
	fmt.Printf("  %-20s%s\n", "Authors staged", formatCount(stats.AuthorsStaged))
	if resolved.SkipEditions {
		fmt.Printf("  %-20s%s\n", "Editions", dimStyle.Render("skipped"))
	} else {
		fmt.Printf("  %-20s%s\n", "Editions matched", formatCount(stats.EditionsStaged))
	}
	fmt.Printf("  %-20s%s\n", "Books imported", formatCount(stats.BooksInserted))
	fmt.Printf("  %-20s%s\n", "Total time", stats.Duration.Round(100*time.Millisecond).String())
	if resolved.SkipEditions {
		fmt.Println(dimStyle.Render("  (editions skipped: ISBN/publisher/page metadata may be missing)"))
	}

	if exportParquet {
		fmt.Println()
		paths, err := openlibrarydump.ExportParquet(ctx, dbPath, parquetDir)
		if err != nil {
			return err
		}
		fmt.Println(successStyle.Render("  Parquet export complete"))
		for _, p := range paths {
			fmt.Printf("    %s\n", p)
		}
	}

	if cleanupSource {
		if err := openlibrarydump.DeleteSourceFiles(resolved.AuthorsPath, resolved.WorksPath, resolved.EditionsPath); err != nil {
			return err
		}
		fmt.Println(successStyle.Render("  Removed source dump files"))
	}
	fmt.Println()
	return nil
}

func formatCount(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	offset := len(s) % 3
	if offset > 0 {
		b.WriteString(s[:offset])
	}
	for i := offset; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func newImportOpenLibrary() *cobra.Command {
	var (
		dir            string
		authorsPath    string
		worksPath      string
		editionsPath   string
		parquetDir     string
		limitWorks     int
		replaceBooks   bool
		skipEditions   bool
		exportParquet  bool
		cleanupSource  bool
		downloadLatest bool
	)

	home, _ := os.UserHomeDir()
	defaultDir := filepath.Join(home, "data", "openlibrary")

	cmd := &cobra.Command{
		Use:   "openlibrary",
		Short: "Download latest Open Library dumps and import into DuckDB",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if downloadLatest {
				fmt.Println(infoStyle.Render("  Resolving latest Open Library dumps..."))
				specs, err := openlibrarydump.ResolveLatestDumpSpecs(ctx)
				if err != nil {
					return err
				}

				var totalSize int64
				for _, spec := range specs {
					totalSize += spec.SizeBytes
				}
				fmt.Println(successStyle.Render("  Found latest dumps"))
				for i, spec := range specs {
					fmt.Printf("  [%d/3] %-10s %s\n", i+1, titleCase(spec.Name), dimStyle.Render(openlibrarydump.FormatBytes(spec.SizeBytes)))
				}
				fmt.Printf("  %-14s%s\n", "Total", dimStyle.Render(openlibrarydump.FormatBytes(totalSize)))

				paths := make(map[string]string, 3)
				for i, spec := range specs {
					fmt.Printf("\n  Downloading [%d/3] %s (%s)\n", i+1, titleCase(spec.Name), openlibrarydump.FormatBytes(spec.SizeBytes))
					path, err := openlibrarydump.DownloadSpec(ctx, spec, dir)
					if err != nil {
						return err
					}
					paths[spec.Name] = path
				}

				authorsPath = paths["authors"]
				worksPath = paths["works"]
				editionsPath = paths["editions"]
			}

			return runOpenLibraryImport(ctx, openlibrarydump.Options{
				Dir:          dir,
				AuthorsPath:  authorsPath,
				WorksPath:    worksPath,
				EditionsPath: editionsPath,
				LimitWorks:   limitWorks,
				ReplaceBooks: replaceBooks,
				SkipEditions: skipEditions,
			}, GetDatabasePath(), parquetDir, exportParquet, cleanupSource)
		},
	}

	cmd.Flags().StringVar(&dir, "dir", defaultDir, "Directory containing Open Library dumps")
	cmd.Flags().StringVar(&authorsPath, "authors", "", "Path to authors dump .txt.gz")
	cmd.Flags().StringVar(&worksPath, "works", "", "Path to works dump .txt.gz")
	cmd.Flags().StringVar(&editionsPath, "editions", "", "Path to editions dump .txt.gz")
	cmd.Flags().StringVar(&parquetDir, "parquet-dir", filepath.Join(defaultDir, "parquet"), "Output directory for parquet exports")
	cmd.Flags().IntVar(&limitWorks, "limit", 0, "Limit number of works to import (0 = no limit)")
	cmd.Flags().BoolVar(&replaceBooks, "replace", true, "Replace existing books with same Open Library key")
	cmd.Flags().BoolVar(&skipEditions, "skip-editions", false, "Skip editions metadata import (faster, less memory/disk usage)")
	cmd.Flags().BoolVar(&exportParquet, "export-parquet", true, "Export imported Open Library records to parquet files")
	cmd.Flags().BoolVar(&cleanupSource, "cleanup-source", true, "Delete source dump files after successful import and parquet export")
	cmd.Flags().BoolVar(&downloadLatest, "download-latest", true, "Download latest dump files before importing")
	return cmd
}

func newImportCSV() *cobra.Command {
	return &cobra.Command{
		Use:   "csv <file>",
		Short: "Import books from CSV export",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			store, err := factory.Open(ctx, GetDatabasePath())
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.Ensure(ctx); err != nil {
				return err
			}
			if err := store.Shelf().SeedDefaults(ctx); err != nil {
				return err
			}

			f, err := os.Open(args[0])
			if err != nil {
				return err
			}
			defer f.Close()

			reader := csv.NewReader(f)
			records, err := reader.ReadAll()
			if err != nil {
				return err
			}
			if len(records) < 2 {
				return fmt.Errorf("CSV is empty")
			}

			// Find column indices from header
			header := records[0]
			cols := make(map[string]int)
			for i, h := range header {
				cols[strings.TrimSpace(h)] = i
			}

			imported := 0
			for _, row := range records[1:] {
				book := &types.Book{}
				if i, ok := cols["Title"]; ok && i < len(row) {
					book.Title = row[i]
				}
				if i, ok := cols["Author"]; ok && i < len(row) {
					book.AuthorNames = row[i]
				}
				if i, ok := cols["ISBN13"]; ok && i < len(row) {
					isbn := strings.Trim(row[i], "=\"")
					book.ISBN13 = isbn
				}
				if i, ok := cols["ISBN"]; ok && i < len(row) {
					isbn := strings.Trim(row[i], "=\"")
					book.ISBN10 = isbn
				}
				if i, ok := cols["Number of Pages"]; ok && i < len(row) {
					book.PageCount, _ = strconv.Atoi(row[i])
				}
				if i, ok := cols["Publisher"]; ok && i < len(row) {
					book.Publisher = row[i]
				}
				if i, ok := cols["Year Published"]; ok && i < len(row) {
					book.PublishYear, _ = strconv.Atoi(row[i])
				}
				if i, ok := cols["Average Rating"]; ok && i < len(row) {
					book.AverageRating, _ = strconv.ParseFloat(row[i], 64)
				}

				if book.Title == "" {
					continue
				}

				if err := store.Book().Create(ctx, book); err != nil {
					continue
				}

				// Handle shelf
				if i, ok := cols["Exclusive Shelf"]; ok && i < len(row) {
					shelf, _ := store.Shelf().GetBySlug(ctx, row[i])
					if shelf != nil {
						store.Shelf().AddBook(ctx, shelf.ID, book.ID)
					}
				}

				// Handle rating
				if i, ok := cols["My Rating"]; ok && i < len(row) {
					rating, _ := strconv.Atoi(row[i])
					if rating > 0 {
						review := &types.Review{BookID: book.ID, Rating: rating}
						if j, ok := cols["My Review"]; ok && j < len(row) && row[j] != "" {
							review.Text = row[j]
						}
						if j, ok := cols["Date Read"]; ok && j < len(row) && row[j] != "" {
							if t, err := time.Parse("2006/01/02", row[j]); err == nil {
								review.FinishedAt = &t
							}
						}
						store.Review().Create(ctx, review)
					}
				}

				imported++
				if imported%10 == 0 {
					fmt.Printf("  Imported %d books...\n", imported)
				}
			}

			fmt.Println(successStyle.Render(fmt.Sprintf("Imported %d books", imported)))
			return nil
		},
	}
}

func newExportCSV() *cobra.Command {
	var outFile string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export library as CSV",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			store, err := factory.Open(ctx, GetDatabasePath())
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.Ensure(ctx); err != nil {
				return err
			}

			if outFile == "" {
				outFile = "book_export.csv"
			}

			f, err := os.Create(outFile)
			if err != nil {
				return err
			}
			defer f.Close()

			w := csv.NewWriter(f)
			defer w.Flush()

			// Write CSV header
			w.Write([]string{
				"Title", "Author", "ISBN", "ISBN13", "My Rating", "Average Rating",
				"Publisher", "Number of Pages", "Year Published", "Date Read",
				"Exclusive Shelf", "My Review",
			})

			// Get all books
			result, err := store.Book().Search(ctx, "", 1, 10000)
			if err != nil {
				return err
			}

			for _, book := range result.Books {
				rating := ""
				shelf := ""
				reviewText := ""
				dateRead := ""

				if review, _ := store.Review().GetUserReview(ctx, book.ID); review != nil {
					if review.Rating > 0 {
						rating = strconv.Itoa(review.Rating)
					}
					reviewText = review.Text
					if review.FinishedAt != nil {
						dateRead = review.FinishedAt.Format("2006/01/02")
					}
				}

				shelves, _ := store.Shelf().GetBookShelves(ctx, book.ID)
				for _, sh := range shelves {
					if sh.IsExclusive {
						shelf = sh.Slug
						break
					}
				}

				w.Write([]string{
					book.Title, book.AuthorNames, book.ISBN10, book.ISBN13,
					rating, fmt.Sprintf("%.2f", book.AverageRating),
					book.Publisher, strconv.Itoa(book.PageCount),
					strconv.Itoa(book.PublishYear), dateRead, shelf, reviewText,
				})
			}

			fmt.Println(successStyle.Render(fmt.Sprintf("Exported to %s", outFile)))
			return nil
		},
	}
	cmd.Flags().StringVarP(&outFile, "file", "f", "", "Output file path")
	return cmd
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
