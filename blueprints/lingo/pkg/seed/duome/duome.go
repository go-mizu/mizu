package duome

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// Seeder orchestrates the full download, parse, and import pipeline
type Seeder struct {
	downloader *Downloader
	parser     *Parser
	importer   *Importer
	baseDir    string
	progress   ProgressCallback
}

// SeederOption configures a Seeder
type SeederOption func(*Seeder)

// WithSeederProgress sets the progress callback
func WithSeederProgress(cb ProgressCallback) SeederOption {
	return func(s *Seeder) {
		s.progress = cb
	}
}

// DefaultBaseDir returns the default data directory
func DefaultBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/lingo/duome"
	}
	return filepath.Join(home, "data", "lingo", "duome")
}

// NewSeeder creates a new Seeder
func NewSeeder(db *sql.DB, opts ...SeederOption) *Seeder {
	baseDir := DefaultBaseDir()

	s := &Seeder{
		baseDir: baseDir,
	}

	for _, opt := range opts {
		opt(s)
	}

	// Create components with progress callbacks
	s.downloader = NewDownloader(baseDir, WithProgress(s.progress))
	s.parser = NewParser(baseDir, WithParserProgress(s.progress))
	s.importer = NewImporter(db, s.parser, WithImporterProgress(s.progress))

	return s
}

// NewSeederWithBaseDir creates a new Seeder with a custom base directory
func NewSeederWithBaseDir(db *sql.DB, baseDir string, opts ...SeederOption) *Seeder {
	s := &Seeder{
		baseDir: baseDir,
	}

	for _, opt := range opts {
		opt(s)
	}

	s.downloader = NewDownloader(baseDir, WithProgress(s.progress))
	s.parser = NewParser(baseDir, WithParserProgress(s.progress))
	s.importer = NewImporter(db, s.parser, WithImporterProgress(s.progress))

	return s
}

// BaseDir returns the base data directory
func (s *Seeder) BaseDir() string {
	return s.baseDir
}

// Downloader returns the downloader component
func (s *Seeder) Downloader() *Downloader {
	return s.downloader
}

// Parser returns the parser component
func (s *Seeder) Parser() *Parser {
	return s.parser
}

// Importer returns the importer component
func (s *Seeder) Importer() *Importer {
	return s.importer
}

// Download downloads HTML files for specified language pairs
func (s *Seeder) Download(ctx context.Context, pairs []LanguagePair) error {
	return s.downloader.DownloadAll(ctx, pairs)
}

// DownloadPair downloads HTML files for a single language pair
func (s *Seeder) DownloadPair(ctx context.Context, pair LanguagePair) error {
	return s.downloader.DownloadPair(ctx, pair)
}

// Parse parses downloaded HTML files
func (s *Seeder) Parse(pairs []LanguagePair) (map[string]*CourseData, error) {
	return s.parser.ParseAll(pairs)
}

// ParsePair parses HTML files for a single language pair
func (s *Seeder) ParsePair(pair LanguagePair) (*CourseData, *TipsData, error) {
	return s.parser.ParsePair(pair)
}

// Import imports parsed data to the database
func (s *Seeder) Import(ctx context.Context, pairs []LanguagePair) error {
	return s.importer.ImportAll(ctx, pairs)
}

// ImportPair imports a single language pair to the database
func (s *Seeder) ImportPair(ctx context.Context, pair LanguagePair) error {
	return s.importer.ImportPair(ctx, pair)
}

// SeedPair performs download, parse, and import for a single language pair
func (s *Seeder) SeedPair(ctx context.Context, pair LanguagePair) error {
	// Download
	if s.progress != nil {
		s.progress(1, 3, fmt.Sprintf("Downloading %s", pair))
	}
	if err := s.DownloadPair(ctx, pair); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	// Parse
	if s.progress != nil {
		s.progress(2, 3, fmt.Sprintf("Parsing %s", pair))
	}
	if _, _, err := s.ParsePair(pair); err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// Import
	if s.progress != nil {
		s.progress(3, 3, fmt.Sprintf("Importing %s", pair))
	}
	if err := s.ImportPair(ctx, pair); err != nil {
		return fmt.Errorf("import: %w", err)
	}

	return nil
}

// SeedAll performs the full pipeline for all supported language pairs
func (s *Seeder) SeedAll(ctx context.Context) error {
	return s.Seed(ctx, GetSupportedPairs())
}

// SeedPrimary performs the full pipeline for primary language pairs
func (s *Seeder) SeedPrimary(ctx context.Context) error {
	return s.Seed(ctx, GetPrimaryPairs())
}

// Seed performs the full pipeline for specified language pairs
func (s *Seeder) Seed(ctx context.Context, pairs []LanguagePair) error {
	total := len(pairs)

	// Download all
	if s.progress != nil {
		s.progress(0, total*3, "Downloading...")
	}
	for i, pair := range pairs {
		if s.progress != nil {
			s.progress(i+1, total*3, fmt.Sprintf("Downloading %s", pair))
		}
		if err := s.DownloadPair(ctx, pair); err != nil {
			fmt.Printf("Warning: failed to download %s: %v\n", pair, err)
		}
	}

	// Parse all
	for i, pair := range pairs {
		if s.progress != nil {
			s.progress(total+i+1, total*3, fmt.Sprintf("Parsing %s", pair))
		}
		if _, _, err := s.ParsePair(pair); err != nil {
			fmt.Printf("Warning: failed to parse %s: %v\n", pair, err)
		}
	}

	// Import all
	for i, pair := range pairs {
		if s.progress != nil {
			s.progress(total*2+i+1, total*3, fmt.Sprintf("Importing %s", pair))
		}
		if err := s.ImportPair(ctx, pair); err != nil {
			fmt.Printf("Warning: failed to import %s: %v\n", pair, err)
		}
	}

	return nil
}

// GetStats returns statistics for all imported courses
func (s *Seeder) GetStats(ctx context.Context, pairs []LanguagePair) (map[string]map[string]int, error) {
	allStats := make(map[string]map[string]int)

	for _, pair := range pairs {
		stats, err := s.importer.GetCourseStats(ctx, pair)
		if err != nil {
			continue // Skip pairs that aren't imported
		}
		allStats[pair.String()] = stats
	}

	return allStats, nil
}

// PrintStats prints statistics for all imported courses
func (s *Seeder) PrintStats(ctx context.Context, pairs []LanguagePair) {
	stats, _ := s.GetStats(ctx, pairs)

	fmt.Println("\nCourse Statistics:")
	fmt.Println("==================")

	for pairStr, pairStats := range stats {
		fmt.Printf("\n%s:\n", pairStr)
		fmt.Printf("  Units:     %d\n", pairStats["units"])
		fmt.Printf("  Skills:    %d\n", pairStats["skills"])
		fmt.Printf("  Lessons:   %d\n", pairStats["lessons"])
		fmt.Printf("  Exercises: %d\n", pairStats["exercises"])
		fmt.Printf("  Lexemes:   %d\n", pairStats["lexemes"])
	}
}

// DownloadStoriesAll downloads ALL stories for a language pair in parallel
func (s *Seeder) DownloadStoriesAll(ctx context.Context, pair LanguagePair, workers int) (int, error) {
	// Download stories list first
	if err := s.downloader.DownloadStoriesList(ctx, pair); err != nil {
		return 0, fmt.Errorf("download stories list: %w", err)
	}

	// Parse stories list to get individual story IDs
	list, err := s.parser.ParseStoriesList(pair)
	if err != nil {
		return 0, fmt.Errorf("parse stories list: %w", err)
	}

	if len(list.Stories) == 0 {
		return 0, nil
	}

	// Collect all story IDs
	storyIDs := make([]string, len(list.Stories))
	for i, story := range list.Stories {
		storyIDs[i] = story.ExternalID
	}

	// Download all stories in parallel
	progressFn := func(done, total, skipped int, current string) {
		fmt.Printf("\r    Downloading [%d/%d] (cached: %d) %s                    ", done, total, skipped, truncate(current, 40))
	}

	if err := s.downloader.DownloadStoriesParallel(ctx, storyIDs, workers, progressFn); err != nil {
		return 0, fmt.Errorf("download stories parallel: %w", err)
	}
	fmt.Println()

	return len(storyIDs), nil
}

// ParseStoriesAll parses ALL downloaded stories for a language pair in parallel
func (s *Seeder) ParseStoriesAll(pair LanguagePair, workers int) ([]*StoryData, error) {
	// Get story IDs from the list
	list, err := s.parser.ParseStoriesList(pair)
	if err != nil {
		return nil, fmt.Errorf("parse stories list: %w", err)
	}

	// Save the list
	if err := s.parser.SaveStoriesListJSON(list, pair); err != nil {
		return nil, fmt.Errorf("save stories list: %w", err)
	}

	if len(list.Stories) == 0 {
		return nil, nil
	}

	// Collect all story IDs
	storyIDs := make([]string, len(list.Stories))
	listItemMap := make(map[string]StoryListItem)
	for i, item := range list.Stories {
		storyIDs[i] = item.ExternalID
		listItemMap[item.ExternalID] = item
	}

	// Parse all stories in parallel
	progressFn := func(done, total int, current string) {
		fmt.Printf("\r    Parsing [%d/%d] %s                    ", done, total, truncate(current, 50))
	}

	stories, err := s.parser.ParseStoriesParallel(storyIDs, pair, workers, progressFn)
	if err != nil {
		return nil, fmt.Errorf("parse stories parallel: %w", err)
	}
	fmt.Println()

	// Enrich stories with metadata from list
	for _, story := range stories {
		if item, ok := listItemMap[story.ExternalID]; ok {
			if story.Title == "" {
				story.Title = item.Title
			}
			if story.CEFRLevel == "" {
				story.CEFRLevel = item.CEFRLevel
			}
			if story.IllustrationURL == "" {
				story.IllustrationURL = item.IllustrationURL
			}
		}
	}

	return stories, nil
}

// ImportStoriesAll imports parsed stories for a language pair (idempotent)
func (s *Seeder) ImportStoriesAll(ctx context.Context, pair LanguagePair, stories []*StoryData) error {
	// Get course ID
	courseID, err := s.getCourseID(ctx, pair)
	if err != nil {
		return fmt.Errorf("get course ID: %w", err)
	}

	return s.importer.ImportStoriesForCourse(ctx, courseID, stories)
}

// getCourseID returns the course ID for a language pair
func (s *Seeder) getCourseID(ctx context.Context, pair LanguagePair) (string, error) {
	return s.importer.GetCourseID(ctx, pair)
}

// SeedStoriesAll performs download, parse, and import of ALL stories for a language pair
func (s *Seeder) SeedStoriesAll(ctx context.Context, pair LanguagePair, workers int) (int, error) {
	fmt.Printf("  %s: Downloading stories...\n", pair)
	count, err := s.DownloadStoriesAll(ctx, pair, workers)
	if err != nil {
		return 0, fmt.Errorf("download stories: %w", err)
	}

	if count == 0 {
		fmt.Printf("  %s: No stories available\n", pair)
		return 0, nil
	}

	fmt.Printf("  %s: Parsing %d stories...\n", pair, count)
	stories, err := s.ParseStoriesAll(pair, workers)
	if err != nil {
		return 0, fmt.Errorf("parse stories: %w", err)
	}

	fmt.Printf("  %s: Importing %d stories...\n", pair, len(stories))
	if err := s.ImportStoriesAll(ctx, pair, stories); err != nil {
		return 0, fmt.Errorf("import stories: %w", err)
	}

	fmt.Printf("  %s: Done! Imported %d stories\n", pair, len(stories))
	return len(stories), nil
}

// truncate shortens a string to a maximum length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
