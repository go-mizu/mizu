package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-mizu/mizu/blueprints/lingo/pkg/seed/duome"
	"github.com/go-mizu/mizu/blueprints/lingo/store/sqlite"
	"github.com/spf13/cobra"
)

// NewSeedDuome creates the seed-duome command
func NewSeedDuome() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed-duome",
		Short: "Seed database from Duome.eu vocabulary data",
		Long: `Download vocabulary and tips from Duome.eu and import into the database.

This command fetches real vocabulary data from Duome.eu for language learning courses.
Supports 20+ language pairs from English.

Examples:
  lingo seed-duome all                 # Full pipeline for primary languages
  lingo seed-duome download --lang ja  # Download Japanese only
  lingo seed-duome parse --lang ja     # Parse downloaded Japanese files
  lingo seed-duome import --lang ja    # Import parsed Japanese data`,
	}

	cmd.AddCommand(
		newSeedDuomeDownload(),
		newSeedDuomeDownloadAll(),
		newSeedDuomeParse(),
		newSeedDuomeImport(),
		newSeedDuomeAll(),
		newSeedDuomeList(),
	)

	return cmd
}

func newSeedDuomeDownload() *cobra.Command {
	var lang string

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download vocabulary and tips HTML files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSeedDuomeDownload(cmd.Context(), lang)
		},
	}

	cmd.Flags().StringVarP(&lang, "lang", "l", "", "Target language code (e.g., ja, es, fr)")
	cmd.MarkFlagRequired("lang")

	return cmd
}

func newSeedDuomeDownloadAll() *cobra.Command {
	var primary bool

	cmd := &cobra.Command{
		Use:   "download-all",
		Short: "Download all supported language pairs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSeedDuomeDownloadAll(cmd.Context(), primary)
		},
	}

	cmd.Flags().BoolVarP(&primary, "primary", "p", true, "Download only primary languages (default: true)")

	return cmd
}

func newSeedDuomeParse() *cobra.Command {
	var lang string

	cmd := &cobra.Command{
		Use:   "parse",
		Short: "Parse downloaded HTML files to JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSeedDuomeParse(lang)
		},
	}

	cmd.Flags().StringVarP(&lang, "lang", "l", "", "Target language code (e.g., ja, es, fr)")
	cmd.MarkFlagRequired("lang")

	return cmd
}

func newSeedDuomeImport() *cobra.Command {
	var lang string
	var useSqlite bool

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import parsed data into database",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSeedDuomeImport(cmd.Context(), lang, useSqlite)
		},
	}

	cmd.Flags().StringVarP(&lang, "lang", "l", "", "Target language code (e.g., ja, es, fr)")
	cmd.Flags().BoolVar(&useSqlite, "sqlite", false, "Use SQLite instead of PostgreSQL")
	cmd.MarkFlagRequired("lang")

	return cmd
}

func newSeedDuomeAll() *cobra.Command {
	var useSqlite bool
	var primary bool

	cmd := &cobra.Command{
		Use:   "all",
		Short: "Run full pipeline: download, parse, and import",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSeedDuomeAll(cmd.Context(), useSqlite, primary)
		},
	}

	cmd.Flags().BoolVar(&useSqlite, "sqlite", false, "Use SQLite instead of PostgreSQL")
	cmd.Flags().BoolVarP(&primary, "primary", "p", true, "Only process primary languages (default: true)")

	return cmd
}

func newSeedDuomeList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List supported languages",
		Run: func(cmd *cobra.Command, args []string) {
			runSeedDuomeList()
		},
	}
}

func runSeedDuomeDownload(ctx context.Context, lang string) error {
	fmt.Println(Banner())
	fmt.Println(infoStyle.Render("Downloading from Duome.eu..."))

	pair := duome.LanguagePair{From: "en", To: lang}

	// Validate language
	if _, ok := duome.SupportedLanguages[lang]; !ok {
		return fmt.Errorf("unsupported language: %s", lang)
	}

	progressCallback := func(current, total int, message string) {
		fmt.Printf("\r%s [%d/%d] %s", infoStyle.Render("  "), current, total, message)
	}

	downloader := duome.NewDownloader(duome.DefaultBaseDir(), duome.WithProgress(progressCallback))

	if err := downloader.DownloadPair(ctx, pair); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("Download complete!"))
	fmt.Printf("  Files saved to: %s\n", downloader.BaseDir())

	return nil
}

func runSeedDuomeDownloadAll(ctx context.Context, primary bool) error {
	fmt.Println(Banner())
	fmt.Println(infoStyle.Render("Downloading all languages from Duome.eu..."))

	var pairs []duome.LanguagePair
	if primary {
		pairs = duome.GetPrimaryPairs()
		fmt.Printf("  Downloading %d primary language pairs\n", len(pairs))
	} else {
		pairs = duome.GetSupportedPairs()
		fmt.Printf("  Downloading %d language pairs\n", len(pairs))
	}

	progressCallback := func(current, total int, message string) {
		fmt.Printf("\r%s [%d/%d] %s          ", infoStyle.Render("  "), current, total, message)
	}

	downloader := duome.NewDownloader(duome.DefaultBaseDir(), duome.WithProgress(progressCallback))

	if err := downloader.DownloadAll(ctx, pairs); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("Download complete!"))
	fmt.Printf("  Files saved to: %s\n", downloader.BaseDir())

	return nil
}

func runSeedDuomeParse(lang string) error {
	fmt.Println(Banner())
	fmt.Println(infoStyle.Render("Parsing Duome data..."))

	pair := duome.LanguagePair{From: "en", To: lang}

	// Validate language
	if _, ok := duome.SupportedLanguages[lang]; !ok {
		return fmt.Errorf("unsupported language: %s", lang)
	}

	parser := duome.NewParser(duome.DefaultBaseDir())

	vocab, tips, err := parser.ParsePair(pair)
	if err != nil {
		return fmt.Errorf("parse failed: %w", err)
	}

	fmt.Println(successStyle.Render("Parse complete!"))
	fmt.Printf("  Skills: %d\n", len(vocab.Skills))
	fmt.Printf("  Words: %d\n", vocab.TotalWords)
	fmt.Printf("  Tips sections: %d\n", len(tips.Skills))
	fmt.Printf("  JSON saved to: %s\n", parser.JSONDir())

	return nil
}

func runSeedDuomeImport(ctx context.Context, lang string, useSqlite bool) error {
	fmt.Println(Banner())
	fmt.Println(infoStyle.Render("Importing Duome data..."))

	pair := duome.LanguagePair{From: "en", To: lang}

	// Validate language
	if _, ok := duome.SupportedLanguages[lang]; !ok {
		return fmt.Errorf("unsupported language: %s", lang)
	}

	var db *sql.DB
	var err error

	if useSqlite {
		dbPath := defaultDBPath()
		// Ensure database directory exists
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return fmt.Errorf("create db directory: %w", err)
		}
		fmt.Println(infoStyle.Render(fmt.Sprintf("Connecting to SQLite (%s)...", dbPath)))
		store, err := sqlite.New(ctx, dbPath)
		if err != nil {
			return fmt.Errorf("connect to sqlite: %w", err)
		}
		defer store.Close()

		if err := store.Ensure(ctx); err != nil {
			return fmt.Errorf("ensure schema: %w", err)
		}

		// Get underlying db connection
		db = store.DB()
	} else {
		fmt.Println(infoStyle.Render("Connecting to PostgreSQL..."))
		db, err = sql.Open("pgx", GetDatabaseURL())
		if err != nil {
			return fmt.Errorf("connect to postgres: %w", err)
		}
		defer db.Close()
	}

	parser := duome.NewParser(duome.DefaultBaseDir())
	importer := duome.NewImporter(db, parser)

	if err := importer.ImportPair(ctx, pair); err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	fmt.Println(successStyle.Render("Import complete!"))

	// Print stats
	stats, err := importer.GetCourseStats(ctx, pair)
	if err == nil {
		fmt.Printf("  Units: %d\n", stats["units"])
		fmt.Printf("  Skills: %d\n", stats["skills"])
		fmt.Printf("  Lessons: %d\n", stats["lessons"])
		fmt.Printf("  Exercises: %d\n", stats["exercises"])
		fmt.Printf("  Lexemes: %d\n", stats["lexemes"])
	}

	return nil
}

func runSeedDuomeAll(ctx context.Context, useSqlite bool, primary bool) error {
	fmt.Println(Banner())
	fmt.Println(infoStyle.Render("Running full Duome seed pipeline..."))

	var pairs []duome.LanguagePair
	if primary {
		pairs = duome.GetPrimaryPairs()
		fmt.Printf("  Processing %d primary language pairs\n", len(pairs))
	} else {
		pairs = duome.GetSupportedPairs()
		fmt.Printf("  Processing %d language pairs\n", len(pairs))
	}

	var db *sql.DB
	var err error

	if useSqlite {
		dbPath := defaultDBPath()
		// Ensure database directory exists
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return fmt.Errorf("create db directory: %w", err)
		}
		fmt.Println(infoStyle.Render(fmt.Sprintf("Connecting to SQLite (%s)...", dbPath)))
		store, err := sqlite.New(ctx, dbPath)
		if err != nil {
			return fmt.Errorf("connect to sqlite: %w", err)
		}
		defer store.Close()

		if err := store.Ensure(ctx); err != nil {
			return fmt.Errorf("ensure schema: %w", err)
		}

		// Seed base data
		if err := store.SeedLanguages(ctx); err != nil {
			return fmt.Errorf("seed languages: %w", err)
		}
		if err := store.SeedAchievements(ctx); err != nil {
			return fmt.Errorf("seed achievements: %w", err)
		}
		if err := store.SeedLeagues(ctx); err != nil {
			return fmt.Errorf("seed leagues: %w", err)
		}
		if err := store.SeedUsers(ctx); err != nil {
			return fmt.Errorf("seed users: %w", err)
		}

		db = store.DB()
	} else {
		fmt.Println(infoStyle.Render("Connecting to PostgreSQL..."))
		db, err = sql.Open("pgx", GetDatabaseURL())
		if err != nil {
			return fmt.Errorf("connect to postgres: %w", err)
		}
		defer db.Close()
	}

	progressCallback := func(current, total int, message string) {
		fmt.Printf("\r%s [%d/%d] %s          ", infoStyle.Render("  "), current, total, message)
	}

	seeder := duome.NewSeederWithBaseDir(db, duome.DefaultBaseDir(), duome.WithSeederProgress(progressCallback))

	if err := seeder.Seed(ctx, pairs); err != nil {
		return fmt.Errorf("seed failed: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("Seed complete!"))
	fmt.Println()

	// Print stats
	seeder.PrintStats(ctx, pairs)

	fmt.Println()
	fmt.Println(subtitleStyle.Render("Sample accounts:"))
	fmt.Println(subtitleStyle.Render("  Email: demo@lingo.dev"))
	fmt.Println(subtitleStyle.Render("  Password: password123"))
	fmt.Println()
	fmt.Println(subtitleStyle.Render("Next step:"))
	fmt.Println(subtitleStyle.Render("  lingo serve   - Start the server"))
	fmt.Println()

	return nil
}

func runSeedDuomeList() {
	fmt.Println(Banner())
	fmt.Println(infoStyle.Render("Supported Languages:"))
	fmt.Println()

	// Primary languages
	fmt.Println(subtitleStyle.Render("Primary Languages:"))
	primaryPairs := duome.GetPrimaryPairs()
	for _, pair := range primaryPairs {
		lang := duome.SupportedLanguages[pair.To]
		fmt.Printf("  %s %s - %s (%s)\n", lang.FlagEmoji, lang.Code, lang.Name, lang.NativeName)
	}
	fmt.Println()

	// All other languages
	fmt.Println(subtitleStyle.Render("Other Languages:"))
	primaryMap := make(map[string]bool)
	for _, pair := range primaryPairs {
		primaryMap[pair.To] = true
	}

	var others []string
	for code := range duome.SupportedLanguages {
		if !primaryMap[code] {
			others = append(others, code)
		}
	}

	for _, code := range others {
		lang := duome.SupportedLanguages[code]
		fmt.Printf("  %s %s - %s (%s)\n", lang.FlagEmoji, lang.Code, lang.Name, lang.NativeName)
	}
	fmt.Println()

	fmt.Printf("Total: %d languages supported\n", len(duome.SupportedLanguages))
	fmt.Println()
	fmt.Println(subtitleStyle.Render("Usage:"))
	fmt.Println(subtitleStyle.Render("  lingo seed-duome download -l ja   # Download Japanese"))
	fmt.Println(subtitleStyle.Render("  lingo seed-duome all --sqlite     # Full pipeline"))
	fmt.Println()
}

// Add method to sqlite store to get underlying DB
func init() {
	// Patch to add DB() method would go here if needed
	// For now, we'll add it to the sqlite store
}

// Helper to get list of language codes
func getSupportedLangCodes() []string {
	codes := make([]string, 0, len(duome.SupportedLanguages))
	for code := range duome.SupportedLanguages {
		codes = append(codes, code)
	}
	return codes
}

// ValidateLangCode checks if a language code is supported
func ValidateLangCode(code string) bool {
	_, ok := duome.SupportedLanguages[code]
	return ok
}

// FormatLanguageList returns a formatted string of supported languages
func FormatLanguageList() string {
	var sb strings.Builder
	for code, lang := range duome.SupportedLanguages {
		sb.WriteString(fmt.Sprintf("%s %s - %s\n", lang.FlagEmoji, code, lang.Name))
	}
	return sb.String()
}
