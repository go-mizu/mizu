package cli

import (
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

	totalSteps := 2 // download + import always; export + verify conditional

	cmd := &cobra.Command{
		Use:   "openlibrary",
		Short: "Download latest Open Library dumps and import into DuckDB",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineStart := time.Now()
			ctx := cmd.Context()
			dbPath := GetDatabasePath()

			if exportParquet {
				totalSteps = 4
			}

			// ── Header ──
			fmt.Println()
			fmt.Println("  Open Library Import")
			fmt.Println("  " + strings.Repeat("═", 20))

			// ── Step 1/4: Download ──
			printStep(1, totalSteps, "Download")
			if downloadLatest {
				specs, err := openlibrarydump.ResolveLatestDumpSpecs(ctx)
				if err != nil {
					return err
				}

				var totalSize int64
				for _, spec := range specs {
					totalSize += spec.SizeBytes
				}

				downloaded := make(map[string]string, 3)
				for _, spec := range specs {
					// Check if file already exists before downloading.
					expectedPath := filepath.Join(dir, filepath.Base(spec.ResolvedURL))
					existedBefore := false
					if info, serr := os.Stat(expectedPath); serr == nil && spec.SizeBytes > 0 && info.Size() == spec.SizeBytes {
						existedBefore = true
					}
					path, err := openlibrarydump.DownloadSpec(ctx, spec, dir)
					if err != nil {
						return err
					}
					downloaded[spec.Name] = path
					status := "(downloaded)"
					if existedBefore {
						status = "(exists)"
					}
					fmt.Printf("  %-2s %-10s %-42s %9s  %s\n",
						successStyle.Render("✓"),
						titleCase(spec.Name),
						filepath.Base(path),
						openlibrarydump.FormatBytes(spec.SizeBytes),
						dimStyle.Render(status),
					)
				}
				fmt.Printf("    %-10s %51s\n", "Total", openlibrarydump.FormatBytes(totalSize))

				authorsPath = downloaded["authors"]
				worksPath = downloaded["works"]
				editionsPath = downloaded["editions"]
			} else {
				// Resolve existing files
				opts := openlibrarydump.Options{
					Dir: dir, AuthorsPath: authorsPath, WorksPath: worksPath, EditionsPath: editionsPath,
				}
				resolved, err := openlibrarydump.ResolvePaths(opts)
				if err != nil {
					return err
				}
				authorsPath = resolved.AuthorsPath
				worksPath = resolved.WorksPath
				editionsPath = resolved.EditionsPath

				for _, item := range []struct{ name, path string }{
					{"Authors", authorsPath}, {"Works", worksPath}, {"Editions", editionsPath},
				} {
					sizeStr := ""
					if info, err := os.Stat(item.path); err == nil {
						sizeStr = openlibrarydump.FormatBytes(info.Size())
					}
					fmt.Printf("  %-2s %-10s %-42s %9s  %s\n",
						successStyle.Render("✓"),
						item.name,
						filepath.Base(item.path),
						sizeStr,
						dimStyle.Render("(exists)"),
					)
				}
			}

			// ── Step 2/4: Import to DuckDB ──
			opts := openlibrarydump.Options{
				Dir:          dir,
				AuthorsPath:  authorsPath,
				WorksPath:    worksPath,
				EditionsPath: editionsPath,
				LimitWorks:   limitWorks,
				ReplaceBooks: replaceBooks,
				SkipEditions: skipEditions,
			}
			resolved, err := openlibrarydump.ResolvePaths(opts)
			if err != nil {
				return err
			}

			printStep(2, totalSteps, "Import to DuckDB")
			stats, err := openlibrarydump.ImportToDuckDB(ctx, dbPath, resolved)
			if err != nil {
				return err
			}
			// Config summary after import (stats has the config info)
			fmt.Printf("    %s\n", dimStyle.Render(
				fmt.Sprintf("%s · %d threads · %s",
					stats.MemoryLimit, stats.Threads,
					openlibrarydump.FormatDuration(stats.Duration),
				),
			))

			// ── Step 3/4: Export to Parquet ──
			var parquetPaths []string
			var exportStats *openlibrarydump.ExportStats
			if exportParquet {
				printStep(3, totalSteps, "Export to Parquet")
				exportStart := time.Now()
				parquetPaths, exportStats, err = openlibrarydump.ExportParquet(ctx, dbPath, parquetDir)
				if err != nil {
					return err
				}
				exportDur := time.Since(exportStart)
				if exportStats != nil {
					for i, p := range parquetPaths {
						var rows int
						var label string
						if i == 0 {
							rows = exportStats.BooksExported
							label = "Books"
						} else {
							rows = exportStats.AuthorsExported
							label = "Authors"
						}
						sizeStr := ""
						if info, err := os.Stat(p); err == nil {
							sizeStr = openlibrarydump.FormatBytes(info.Size())
						}
						fmt.Printf("  %-2s %-10s %-28s %12s rows   %9s\n",
							successStyle.Render("✓"),
							label,
							filepath.Base(p),
							openlibrarydump.FormatNumber(rows),
							sizeStr,
						)
					}
					fmt.Printf("    %-52s %s\n", "", openlibrarydump.FormatDuration(exportDur))
				}
			}

			// ── Step 4/4: Verify Parquet ──
			if exportParquet && len(parquetPaths) == 2 {
				printStep(4, totalSteps, "Verify Parquet")
				vs, err := openlibrarydump.VerifyParquet(ctx, parquetPaths[0], parquetPaths[1])
				if err != nil {
					fmt.Printf("  %s %s\n", errorStyle.Render("!"), fmt.Sprintf("verification failed: %v", err))
				} else {
					const barW = 20
					fmt.Printf("  Books (%s rows)\n", openlibrarydump.FormatNumber(vs.BookRows))
					printBar("Title", vs.WithTitle, vs.BookRows, barW)
					printBar("ISBN", vs.WithISBN, vs.BookRows, barW)
					printBar("Cover", vs.WithCover, vs.BookRows, barW)
					printBar("Rated", vs.WithRating, vs.BookRows, barW)
					if vs.AvgRating > 0 {
						fmt.Printf("    %-11s%12.2f\n", "Avg rating", vs.AvgRating)
					}
					fmt.Println()
					fmt.Printf("  Authors (%s rows)\n", openlibrarydump.FormatNumber(vs.AuthorRows))
					printBar("With name", vs.WithName, vs.AuthorRows, barW)
					printBar("With bio", vs.WithBio, vs.AuthorRows, barW)
				}
			}

			// ── Cleanup ──
			if cleanupSource {
				if err := openlibrarydump.DeleteSourceFiles(resolved.AuthorsPath, resolved.WorksPath, resolved.EditionsPath); err != nil {
					return err
				}
			}

			// ── Summary ──
			totalDur := time.Since(pipelineStart)
			summaryLine1 := fmt.Sprintf("✓ %s works → %s books  (%s authors)",
				openlibrarydump.FormatNumber(stats.WorksStaged),
				openlibrarydump.FormatNumber(stats.BooksInserted),
				openlibrarydump.FormatNumber(stats.AuthorsStaged),
			)
			var summaryLine2 string
			if exportParquet && exportStats != nil {
				totalPqSize := int64(0)
				for _, p := range parquetPaths {
					if info, err := os.Stat(p); err == nil {
						totalPqSize += info.Size()
					}
				}
				summaryLine2 = fmt.Sprintf("Parquet: %d files (%s)", len(parquetPaths), openlibrarydump.FormatBytes(totalPqSize))
			}
			summaryLine3 := fmt.Sprintf("Total time: %s", openlibrarydump.FormatDuration(totalDur))

			if summaryLine2 != "" {
				printSummaryBox(summaryLine1, summaryLine2, summaryLine3)
			} else {
				printSummaryBox(summaryLine1, summaryLine3)
			}
			fmt.Println()
			return nil
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
	cmd.Flags().BoolVar(&cleanupSource, "cleanup-source", false, "Delete source dump files after successful import and parquet export")
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
