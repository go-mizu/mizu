package duome

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/html"
)

func TestGetSupportedPairs(t *testing.T) {
	pairs := GetSupportedPairs()
	if len(pairs) == 0 {
		t.Error("expected at least one supported pair")
	}

	// Verify we have pairs from multiple source languages
	sourceLanguages := make(map[string]bool)
	for _, pair := range pairs {
		sourceLanguages[pair.From] = true
	}
	if len(sourceLanguages) < 10 {
		t.Errorf("expected at least 10 source languages, got %d", len(sourceLanguages))
	}
}

func TestGetPairsFromEnglish(t *testing.T) {
	pairs := GetPairsFromEnglish()
	if len(pairs) == 0 {
		t.Error("expected at least one pair from English")
	}

	// Verify all pairs are from English
	for _, pair := range pairs {
		if pair.From != "en" {
			t.Errorf("expected From to be 'en', got %s", pair.From)
		}
	}
}

func TestGetPrimaryPairs(t *testing.T) {
	pairs := GetPrimaryPairs()
	if len(pairs) != 10 {
		t.Errorf("expected 10 primary pairs, got %d", len(pairs))
	}

	// Check that Japanese is included
	hasJapanese := false
	for _, pair := range pairs {
		if pair.To == "ja" {
			hasJapanese = true
			break
		}
	}
	if !hasJapanese {
		t.Error("expected Japanese to be in primary pairs")
	}
}

func TestLanguagePairString(t *testing.T) {
	pair := LanguagePair{From: "en", To: "ja"}
	if pair.String() != "en/ja" {
		t.Errorf("expected 'en/ja', got %s", pair.String())
	}
}

func TestLanguagePairURLs(t *testing.T) {
	pair := LanguagePair{From: "en", To: "ja"}

	vocabURL := pair.VocabularyURL()
	if vocabURL != "https://duome.eu/vocabulary/en/ja/skills" {
		t.Errorf("unexpected vocabulary URL: %s", vocabURL)
	}

	tipsURL := pair.TipsURL()
	if tipsURL != "https://duome.eu/tips/en/ja" {
		t.Errorf("unexpected tips URL: %s", tipsURL)
	}
}

func TestParseVocabularyHTML(t *testing.T) {
	// Read test fixture
	testData, err := os.ReadFile("testdata/vocabulary_en_ja_sample.html")
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	doc, err := html.Parse(bytes.NewReader(testData))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}

	entries := extractVocabularyEntries(doc)

	if len(entries) != 8 {
		t.Errorf("expected 8 entries, got %d", len(entries))
	}

	// Check first entry
	if len(entries) > 0 {
		first := entries[0]
		if first.Word != "おちゃ" {
			t.Errorf("expected first word 'おちゃ', got '%s'", first.Word)
		}
		if first.Romanization != "ocha" {
			t.Errorf("expected romanization 'ocha', got '%s'", first.Romanization)
		}
		if len(first.Translations) < 1 || first.Translations[0] != "green tea" {
			t.Errorf("expected first translation 'green tea', got %v", first.Translations)
		}
		if first.SkillName != "Basics" {
			t.Errorf("expected skill 'Basics', got '%s'", first.SkillName)
		}
	}
}

func TestParseSkillHeader(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Basics · 11 · 2024-11-08", "Basics"},
		{"Greetings · 8 · 2024-11-08", "Greetings"},
		{"Simple Text", "Simple Text"},
		{"Food · 15", "Food"},
	}

	for _, tc := range tests {
		result := parseSkillHeader(tc.input)
		if result != tc.expected {
			t.Errorf("parseSkillHeader(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestParserWithFixtures(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	rawDir := filepath.Join(tmpDir, "raw")
	if err := os.MkdirAll(rawDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Copy test fixtures
	vocabData, err := os.ReadFile("testdata/vocabulary_en_ja_sample.html")
	if err != nil {
		t.Fatalf("failed to read vocabulary fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rawDir, "vocabulary_en_ja.html"), vocabData, 0644); err != nil {
		t.Fatalf("failed to write vocabulary fixture: %v", err)
	}

	tipsData, err := os.ReadFile("testdata/tips_en_ja_sample.html")
	if err != nil {
		t.Fatalf("failed to read tips fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rawDir, "tips_en_ja.html"), tipsData, 0644); err != nil {
		t.Fatalf("failed to write tips fixture: %v", err)
	}

	// Test parser
	parser := NewParser(tmpDir)
	pair := LanguagePair{From: "en", To: "ja"}

	// Parse vocabulary
	courseData, err := parser.ParseVocabulary(pair)
	if err != nil {
		t.Fatalf("ParseVocabulary failed: %v", err)
	}

	if courseData.FromLanguage != "en" {
		t.Errorf("expected FromLanguage 'en', got '%s'", courseData.FromLanguage)
	}
	if courseData.ToLanguage != "ja" {
		t.Errorf("expected ToLanguage 'ja', got '%s'", courseData.ToLanguage)
	}
	if courseData.TotalWords != 8 {
		t.Errorf("expected 8 total words, got %d", courseData.TotalWords)
	}
	if len(courseData.Skills) != 3 {
		t.Errorf("expected 3 skills, got %d", len(courseData.Skills))
	}

	// Check skills
	skillNames := make(map[string]bool)
	for _, skill := range courseData.Skills {
		skillNames[skill.Name] = true
	}
	if !skillNames["Basics"] || !skillNames["Food"] || !skillNames["Greetings"] {
		t.Errorf("expected skills Basics, Food, Greetings, got %v", skillNames)
	}

	// Parse tips
	tipsDataParsed, err := parser.ParseTips(pair)
	if err != nil {
		t.Fatalf("ParseTips failed: %v", err)
	}

	if len(tipsDataParsed.Skills) != 3 {
		t.Errorf("expected 3 tip sections, got %d", len(tipsDataParsed.Skills))
	}

	// Check Basics tips
	basicsTips, ok := tipsDataParsed.Skills["Basics"]
	if !ok {
		t.Error("expected Basics tips to exist")
	} else {
		if len(basicsTips.Tables) == 0 {
			t.Error("expected Basics to have tables")
		}
		if basicsTips.Content == "" {
			t.Error("expected Basics to have content")
		}
	}
}

func TestParserParseAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	rawDir := filepath.Join(tmpDir, "raw")
	jsonDir := filepath.Join(tmpDir, "json")
	if err := os.MkdirAll(rawDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Copy test fixtures
	vocabData, err := os.ReadFile("testdata/vocabulary_en_ja_sample.html")
	if err != nil {
		t.Fatalf("failed to read vocabulary fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rawDir, "vocabulary_en_ja.html"), vocabData, 0644); err != nil {
		t.Fatalf("failed to write vocabulary fixture: %v", err)
	}

	parser := NewParser(tmpDir)
	pair := LanguagePair{From: "en", To: "ja"}

	// Parse and save
	courseData, err := parser.ParseAndSaveVocabulary(pair)
	if err != nil {
		t.Fatalf("ParseAndSaveVocabulary failed: %v", err)
	}

	// Check JSON file was created
	jsonPath := filepath.Join(jsonDir, "vocabulary_en_ja.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("expected JSON file to be created")
	}

	// Load from JSON
	loadedData, err := parser.LoadVocabularyJSON(pair)
	if err != nil {
		t.Fatalf("LoadVocabularyJSON failed: %v", err)
	}

	if loadedData.TotalWords != courseData.TotalWords {
		t.Errorf("loaded data mismatch: expected %d words, got %d", courseData.TotalWords, loadedData.TotalWords)
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:?cache=shared&_foreign_keys=on")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Create tables
	statements := []string{
		`CREATE TABLE IF NOT EXISTS languages (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			native_name TEXT,
			flag_emoji TEXT,
			rtl INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS courses (
			id TEXT PRIMARY KEY,
			from_language_id TEXT,
			learning_language_id TEXT,
			title TEXT,
			description TEXT,
			total_units INTEGER DEFAULT 0,
			cefr_level TEXT,
			enabled INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS units (
			id TEXT PRIMARY KEY,
			course_id TEXT,
			position INTEGER NOT NULL,
			title TEXT,
			description TEXT,
			guidebook_content TEXT,
			icon_url TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS skills (
			id TEXT PRIMARY KEY,
			unit_id TEXT,
			position INTEGER NOT NULL,
			name TEXT,
			icon_name TEXT,
			levels INTEGER DEFAULT 5,
			lexemes_count INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS lessons (
			id TEXT PRIMARY KEY,
			skill_id TEXT,
			level INTEGER NOT NULL,
			position INTEGER NOT NULL,
			exercise_count INTEGER DEFAULT 15
		)`,
		`CREATE TABLE IF NOT EXISTS exercises (
			id TEXT PRIMARY KEY,
			lesson_id TEXT,
			type TEXT NOT NULL,
			prompt TEXT,
			correct_answer TEXT,
			choices TEXT,
			audio_url TEXT,
			image_url TEXT,
			hints TEXT,
			difficulty INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS lexemes (
			id TEXT PRIMARY KEY,
			course_id TEXT,
			word TEXT NOT NULL,
			translation TEXT,
			pos TEXT,
			gender TEXT,
			audio_url TEXT,
			image_url TEXT,
			example_sentence TEXT,
			example_translation TEXT
		)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}

	return db
}

func TestImporter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	rawDir := filepath.Join(tmpDir, "raw")
	jsonDir := filepath.Join(tmpDir, "json")
	if err := os.MkdirAll(rawDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(jsonDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Copy and parse fixtures
	vocabData, err := os.ReadFile("testdata/vocabulary_en_ja_sample.html")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rawDir, "vocabulary_en_ja.html"), vocabData, 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	parser := NewParser(tmpDir)
	pair := LanguagePair{From: "en", To: "ja"}

	// Parse and save
	_, err = parser.ParseAndSaveVocabulary(pair)
	if err != nil {
		t.Fatalf("ParseAndSaveVocabulary failed: %v", err)
	}

	// Import
	importer := NewImporter(db, parser)
	ctx := context.Background()

	if err := importer.ImportPair(ctx, pair); err != nil {
		t.Fatalf("ImportPair failed: %v", err)
	}

	// Verify languages
	var langCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM languages").Scan(&langCount); err != nil {
		t.Fatalf("failed to count languages: %v", err)
	}
	if langCount != 2 { // en and ja
		t.Errorf("expected 2 languages, got %d", langCount)
	}

	// Verify course
	var courseCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM courses").Scan(&courseCount); err != nil {
		t.Fatalf("failed to count courses: %v", err)
	}
	if courseCount != 1 {
		t.Errorf("expected 1 course, got %d", courseCount)
	}

	// Verify units
	var unitCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM units").Scan(&unitCount); err != nil {
		t.Fatalf("failed to count units: %v", err)
	}
	if unitCount == 0 {
		t.Error("expected at least 1 unit")
	}

	// Verify skills
	var skillCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM skills").Scan(&skillCount); err != nil {
		t.Fatalf("failed to count skills: %v", err)
	}
	if skillCount != 3 { // Basics, Food, Greetings
		t.Errorf("expected 3 skills, got %d", skillCount)
	}

	// Verify lessons
	var lessonCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM lessons").Scan(&lessonCount); err != nil {
		t.Fatalf("failed to count lessons: %v", err)
	}
	if lessonCount != 15 { // 3 skills * 5 levels
		t.Errorf("expected 15 lessons, got %d", lessonCount)
	}

	// Verify exercises
	var exerciseCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM exercises").Scan(&exerciseCount); err != nil {
		t.Fatalf("failed to count exercises: %v", err)
	}
	if exerciseCount != 225 { // 15 lessons * 15 exercises
		t.Errorf("expected 225 exercises, got %d", exerciseCount)
	}

	// Verify lexemes
	var lexemeCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM lexemes").Scan(&lexemeCount); err != nil {
		t.Fatalf("failed to count lexemes: %v", err)
	}
	if lexemeCount != 8 {
		t.Errorf("expected 8 lexemes, got %d", lexemeCount)
	}

	// Test GetCourseStats
	stats, err := importer.GetCourseStats(ctx, pair)
	if err != nil {
		t.Fatalf("GetCourseStats failed: %v", err)
	}
	if stats["units"] != unitCount {
		t.Errorf("stats mismatch: units expected %d, got %d", unitCount, stats["units"])
	}
	if stats["skills"] != skillCount {
		t.Errorf("stats mismatch: skills expected %d, got %d", skillCount, stats["skills"])
	}
}

func TestSeeder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()

	// Setup fixtures
	rawDir := filepath.Join(tmpDir, "raw")
	if err := os.MkdirAll(rawDir, 0755); err != nil {
		t.Fatal(err)
	}

	vocabData, err := os.ReadFile("testdata/vocabulary_en_ja_sample.html")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rawDir, "vocabulary_en_ja.html"), vocabData, 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	tipsData, err := os.ReadFile("testdata/tips_en_ja_sample.html")
	if err != nil {
		t.Fatalf("failed to read tips fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rawDir, "tips_en_ja.html"), tipsData, 0644); err != nil {
		t.Fatalf("failed to write tips fixture: %v", err)
	}

	// Create seeder
	var progressCalls int
	progressCallback := func(current, total int, message string) {
		progressCalls++
	}

	seeder := NewSeederWithBaseDir(db, tmpDir, WithSeederProgress(progressCallback))

	// Parse and import (skip download for test)
	ctx := context.Background()
	pair := LanguagePair{From: "en", To: "ja"}

	_, _, err = seeder.ParsePair(pair)
	if err != nil {
		t.Fatalf("ParsePair failed: %v", err)
	}

	err = seeder.ImportPair(ctx, pair)
	if err != nil {
		t.Fatalf("ImportPair failed: %v", err)
	}

	// Verify data
	stats, err := seeder.Importer().GetCourseStats(ctx, pair)
	if err != nil {
		t.Fatalf("GetCourseStats failed: %v", err)
	}

	if stats["skills"] != 3 {
		t.Errorf("expected 3 skills, got %d", stats["skills"])
	}

	// Test progress callback with SeedPair (which does call progress)
	// Copy fixtures again since import deletes old content
	vocabData2, _ := os.ReadFile("testdata/vocabulary_en_ja_sample.html")
	os.WriteFile(filepath.Join(rawDir, "vocabulary_en_ja.html"), vocabData2, 0644)
	tipsData2, _ := os.ReadFile("testdata/tips_en_ja_sample.html")
	os.WriteFile(filepath.Join(rawDir, "tips_en_ja.html"), tipsData2, 0644)

	// Note: SeedPair would try to download, so we test Import batch instead
	pairs := []LanguagePair{pair}
	progressCalls = 0
	seeder.Import(ctx, pairs)

	if progressCalls == 0 {
		t.Error("expected progress callback to be called during batch import")
	}
}

func TestDownloaderPaths(t *testing.T) {
	tmpDir := t.TempDir()
	downloader := NewDownloader(tmpDir)

	pair := LanguagePair{From: "en", To: "ja"}

	vocabPath := downloader.VocabularyPath(pair)
	expectedVocab := filepath.Join(tmpDir, "raw", "vocabulary_en_ja.html")
	if vocabPath != expectedVocab {
		t.Errorf("expected %s, got %s", expectedVocab, vocabPath)
	}

	tipsPath := downloader.TipsPath(pair)
	expectedTips := filepath.Join(tmpDir, "raw", "tips_en_ja.html")
	if tipsPath != expectedTips {
		t.Errorf("expected %s, got %s", expectedTips, tipsPath)
	}
}

func TestExerciseTypeDistribution(t *testing.T) {
	types := make(map[string]int)

	for i := 0; i < 15; i++ {
		for level := 1; level <= 5; level++ {
			exType := getExerciseType(i, level)
			types[exType]++
		}
	}

	// Verify we have multiple exercise types
	if len(types) < 3 {
		t.Errorf("expected at least 3 exercise types, got %d: %v", len(types), types)
	}
}

func TestGetIconForSkill(t *testing.T) {
	tests := []struct {
		skillName string
		expected  string
	}{
		{"Basics", "book"},
		{"Greetings", "hand-wave"},
		{"Family", "users"},
		{"Food", "utensils"},
		{"Travel", "plane"},
		{"Unknown Skill", "book"}, // Default
	}

	for _, tc := range tests {
		icon := getIconForSkill(tc.skillName)
		if icon != tc.expected {
			t.Errorf("getIconForSkill(%q) = %q, want %q", tc.skillName, icon, tc.expected)
		}
	}
}

func TestGenerateDistractors(t *testing.T) {
	vocab := []VocabularyEntry{
		{Word: "word1", Translations: []string{"trans1"}},
		{Word: "word2", Translations: []string{"trans2"}},
		{Word: "word3", Translations: []string{"trans3"}},
		{Word: "word4", Translations: []string{"trans4"}},
		{Word: "word5", Translations: []string{"trans5"}},
	}

	choices := generateDistractors("trans1", vocab, 4)

	if len(choices) != 4 {
		t.Errorf("expected 4 choices, got %d", len(choices))
	}

	// Correct answer should be in choices
	hasCorrect := false
	for _, c := range choices {
		if c == "trans1" {
			hasCorrect = true
			break
		}
	}
	if !hasCorrect {
		t.Error("expected correct answer to be in choices")
	}
}

func TestSupportedLanguagesInfo(t *testing.T) {
	// Check a few specific languages
	ja, ok := SupportedLanguages["ja"]
	if !ok {
		t.Fatal("expected Japanese to be supported")
	}
	if ja.Name != "Japanese" {
		t.Errorf("expected name 'Japanese', got %s", ja.Name)
	}
	if !ja.HasRoman {
		t.Error("expected Japanese to have romanization")
	}

	ar, ok := SupportedLanguages["ar"]
	if !ok {
		t.Fatal("expected Arabic to be supported")
	}
	if !ar.RTL {
		t.Error("expected Arabic to be RTL")
	}

	es, ok := SupportedLanguages["es"]
	if !ok {
		t.Fatal("expected Spanish to be supported")
	}
	if es.RTL {
		t.Error("expected Spanish to not be RTL")
	}
	if es.HasRoman {
		t.Error("expected Spanish to not need romanization")
	}
}
