package duome

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// Importer transforms parsed Duome data into store models
type Importer struct {
	db       *sql.DB
	parser   *Parser
	progress ProgressCallback
}

// ImporterOption configures an Importer
type ImporterOption func(*Importer)

// WithImporterProgress sets the progress callback
func WithImporterProgress(cb ProgressCallback) ImporterOption {
	return func(i *Importer) {
		i.progress = cb
	}
}

// NewImporter creates a new Importer
func NewImporter(db *sql.DB, parser *Parser, opts ...ImporterOption) *Importer {
	i := &Importer{
		db:     db,
		parser: parser,
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// SkillsPerUnit defines how many skills to group into a unit
const SkillsPerUnit = 7

// LessonsPerLevel defines lessons per skill level
const LessonsPerLevel = 1

// ExercisesPerLesson defines exercises per lesson
const ExercisesPerLesson = 15

// ImportPair imports a language pair from JSON files
func (i *Importer) ImportPair(ctx context.Context, pair LanguagePair) error {
	// Load parsed data
	courseData, err := i.parser.LoadVocabularyJSON(pair)
	if err != nil {
		return fmt.Errorf("load vocabulary json: %w", err)
	}

	// Ensure languages exist
	if err := i.ensureLanguages(ctx, pair); err != nil {
		return fmt.Errorf("ensure languages: %w", err)
	}

	// Create or get course
	courseID, err := i.ensureCourse(ctx, pair, courseData)
	if err != nil {
		return fmt.Errorf("ensure course: %w", err)
	}

	// Import content
	if err := i.importCourseContent(ctx, courseID, courseData); err != nil {
		return fmt.Errorf("import course content: %w", err)
	}

	return nil
}

// ensureLanguages ensures both languages exist in the database
func (i *Importer) ensureLanguages(ctx context.Context, pair LanguagePair) error {
	// English (from language)
	_, err := i.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO languages (id, name, native_name, flag_emoji, rtl, enabled)
		VALUES (?, ?, ?, ?, ?, 1)
	`, "en", "English", "English", "🇺🇸", 0)
	if err != nil {
		return fmt.Errorf("insert English: %w", err)
	}

	// Target language
	lang, ok := SupportedLanguages[pair.To]
	if !ok {
		return fmt.Errorf("unsupported language: %s", pair.To)
	}

	rtl := 0
	if lang.RTL {
		rtl = 1
	}

	_, err = i.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO languages (id, name, native_name, flag_emoji, rtl, enabled)
		VALUES (?, ?, ?, ?, ?, 1)
	`, lang.Code, lang.Name, lang.NativeName, lang.FlagEmoji, rtl)
	if err != nil {
		return fmt.Errorf("insert %s: %w", lang.Name, err)
	}

	return nil
}

// ensureCourse creates or returns an existing course
func (i *Importer) ensureCourse(ctx context.Context, pair LanguagePair, data *CourseData) (string, error) {
	lang := SupportedLanguages[pair.To]
	title := fmt.Sprintf("%s for English Speakers", lang.Name)
	description := fmt.Sprintf("Learn %s from scratch with %d vocabulary words across %d skills",
		lang.Name, data.TotalWords, len(data.Skills))

	// Check if course exists
	var existingID string
	err := i.db.QueryRowContext(ctx, `
		SELECT id FROM courses
		WHERE from_language_id = ? AND learning_language_id = ?
	`, pair.From, pair.To).Scan(&existingID)

	if err == nil {
		// Course exists, delete old content for fresh import
		if err := i.deleteCourseContent(ctx, existingID); err != nil {
			return "", fmt.Errorf("delete old content: %w", err)
		}
		return existingID, nil
	}

	if err != sql.ErrNoRows {
		return "", fmt.Errorf("check existing course: %w", err)
	}

	// Create new course
	courseID := uuid.New().String()
	totalUnits := (len(data.Skills) + SkillsPerUnit - 1) / SkillsPerUnit

	_, err = i.db.ExecContext(ctx, `
		INSERT INTO courses (id, from_language_id, learning_language_id, title, description, total_units, cefr_level, enabled)
		VALUES (?, ?, ?, ?, ?, ?, 'A1', 1)
	`, courseID, pair.From, pair.To, title, description, totalUnits)
	if err != nil {
		return "", fmt.Errorf("insert course: %w", err)
	}

	return courseID, nil
}

// deleteCourseContent removes all content for a course
func (i *Importer) deleteCourseContent(ctx context.Context, courseID string) error {
	// Get all unit IDs
	rows, err := i.db.QueryContext(ctx, "SELECT id FROM units WHERE course_id = ?", courseID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var unitIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		unitIDs = append(unitIDs, id)
	}

	// Delete skills, lessons, exercises for each unit
	for _, unitID := range unitIDs {
		// Get skill IDs
		skillRows, err := i.db.QueryContext(ctx, "SELECT id FROM skills WHERE unit_id = ?", unitID)
		if err != nil {
			return err
		}

		var skillIDs []string
		for skillRows.Next() {
			var id string
			if err := skillRows.Scan(&id); err != nil {
				skillRows.Close()
				return err
			}
			skillIDs = append(skillIDs, id)
		}
		skillRows.Close()

		// Delete lessons and exercises for each skill
		for _, skillID := range skillIDs {
			// Get lesson IDs
			lessonRows, err := i.db.QueryContext(ctx, "SELECT id FROM lessons WHERE skill_id = ?", skillID)
			if err != nil {
				return err
			}

			var lessonIDs []string
			for lessonRows.Next() {
				var id string
				if err := lessonRows.Scan(&id); err != nil {
					lessonRows.Close()
					return err
				}
				lessonIDs = append(lessonIDs, id)
			}
			lessonRows.Close()

			// Delete exercises
			for _, lessonID := range lessonIDs {
				_, err := i.db.ExecContext(ctx, "DELETE FROM exercises WHERE lesson_id = ?", lessonID)
				if err != nil {
					return err
				}
			}

			// Delete lessons
			_, err = i.db.ExecContext(ctx, "DELETE FROM lessons WHERE skill_id = ?", skillID)
			if err != nil {
				return err
			}
		}

		// Delete skills
		_, err = i.db.ExecContext(ctx, "DELETE FROM skills WHERE unit_id = ?", unitID)
		if err != nil {
			return err
		}
	}

	// Delete units
	_, err = i.db.ExecContext(ctx, "DELETE FROM units WHERE course_id = ?", courseID)
	if err != nil {
		return err
	}

	// Delete lexemes
	_, err = i.db.ExecContext(ctx, "DELETE FROM lexemes WHERE course_id = ?", courseID)
	if err != nil {
		return err
	}

	return nil
}

// importCourseContent imports all units, skills, lessons, and exercises
func (i *Importer) importCourseContent(ctx context.Context, courseID string, data *CourseData) error {
	// Group skills into units
	unitCount := (len(data.Skills) + SkillsPerUnit - 1) / SkillsPerUnit

	skillIndex := 0
	for unitPos := 1; unitPos <= unitCount; unitPos++ {
		// Determine skills for this unit
		endIndex := skillIndex + SkillsPerUnit
		if endIndex > len(data.Skills) {
			endIndex = len(data.Skills)
		}
		unitSkills := data.Skills[skillIndex:endIndex]

		// Create unit
		unitTitle := fmt.Sprintf("Unit %d", unitPos)
		if len(unitSkills) > 0 {
			// Name unit after first skill
			unitTitle = unitSkills[0].Name
		}

		unitID, err := i.createUnit(ctx, courseID, unitPos, unitTitle, unitSkills)
		if err != nil {
			return fmt.Errorf("create unit %d: %w", unitPos, err)
		}

		// Create skills with language code for audio generation
		for localPos, skill := range unitSkills {
			if err := i.importSkill(ctx, courseID, unitID, localPos+1, skill, data.ToLanguage); err != nil {
				return fmt.Errorf("import skill %s: %w", skill.Name, err)
			}
		}

		skillIndex = endIndex
	}

	return nil
}

// createUnit creates a unit record
func (i *Importer) createUnit(ctx context.Context, courseID string, position int, title string, skills []Skill) (string, error) {
	unitID := uuid.New().String()

	// Build guidebook content from skill tips
	var guidebook strings.Builder
	guidebook.WriteString(fmt.Sprintf("# %s\n\n", title))
	for _, skill := range skills {
		if skill.Tips != nil && skill.Tips.Content != "" {
			guidebook.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", skill.Name, skill.Tips.Content))
		}
	}

	description := fmt.Sprintf("Unit %d contains %d skills", position, len(skills))

	_, err := i.db.ExecContext(ctx, `
		INSERT INTO units (id, course_id, position, title, description, guidebook_content)
		VALUES (?, ?, ?, ?, ?, ?)
	`, unitID, courseID, position, title, description, guidebook.String())
	if err != nil {
		return "", err
	}

	return unitID, nil
}

// importSkill imports a skill with its lessons and exercises
func (i *Importer) importSkill(ctx context.Context, courseID, unitID string, position int, skill Skill, languageCode string) error {
	skillID := uuid.New().String()

	_, err := i.db.ExecContext(ctx, `
		INSERT INTO skills (id, unit_id, position, name, icon_name, levels, lexemes_count)
		VALUES (?, ?, ?, ?, ?, 5, ?)
	`, skillID, unitID, position, skill.Name, getIconForSkill(skill.Name), len(skill.Vocabulary))
	if err != nil {
		return fmt.Errorf("insert skill: %w", err)
	}

	// Import lexemes for this skill with audio URLs
	lexemeIDs := make([]string, 0, len(skill.Vocabulary))
	for _, vocab := range skill.Vocabulary {
		lexemeID, err := i.importLexeme(ctx, courseID, vocab, languageCode)
		if err != nil {
			return fmt.Errorf("import lexeme %s: %w", vocab.Word, err)
		}
		lexemeIDs = append(lexemeIDs, lexemeID)
	}

	// Create lessons (5 levels, 1 lesson per level)
	for level := 1; level <= 5; level++ {
		lessonID := uuid.New().String()

		_, err := i.db.ExecContext(ctx, `
			INSERT INTO lessons (id, skill_id, level, position, exercise_count)
			VALUES (?, ?, ?, ?, ?)
		`, lessonID, skillID, level, 1, ExercisesPerLesson)
		if err != nil {
			return fmt.Errorf("insert lesson: %w", err)
		}

		// Generate exercises from vocabulary with audio support
		if err := i.generateExercises(ctx, lessonID, skill.Vocabulary, level, languageCode); err != nil {
			return fmt.Errorf("generate exercises: %w", err)
		}
	}

	return nil
}

// importLexeme imports a vocabulary word with audio URL
func (i *Importer) importLexeme(ctx context.Context, courseID string, vocab VocabularyEntry, languageCode string) (string, error) {
	lexemeID := uuid.New().String()

	translation := strings.Join(vocab.Translations, ", ")
	if len(translation) > 500 {
		translation = translation[:500]
	}

	// Create example sentence from first translation
	var exampleSentence, exampleTranslation string
	if len(vocab.Translations) > 0 {
		exampleSentence = vocab.Word
		exampleTranslation = vocab.Translations[0]
	}

	// Generate audio URL if not provided
	audioURL := vocab.AudioURL
	if audioURL == "" && LanguageHasAudio(languageCode) {
		audioURL = GenerateAudioURL(vocab.Word, languageCode)
	}

	_, err := i.db.ExecContext(ctx, `
		INSERT INTO lexemes (id, course_id, word, translation, pos, audio_url, example_sentence, example_translation)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, lexemeID, courseID, vocab.Word, translation, vocab.POS, audioURL, exampleSentence, exampleTranslation)
	if err != nil {
		return "", err
	}

	return lexemeID, nil
}

// generateExercises creates exercises for a lesson with audio support
func (i *Importer) generateExercises(ctx context.Context, lessonID string, vocabulary []VocabularyEntry, level int, languageCode string) error {
	if len(vocabulary) == 0 {
		return nil
	}

	exercises := make([]store.Exercise, 0, ExercisesPerLesson)

	// Generate diverse exercise types
	for j := 0; j < ExercisesPerLesson; j++ {
		vocab := vocabulary[j%len(vocabulary)]
		exType := getExerciseType(j, level)

		exercise := generateExercise(vocab, vocabulary, exType, level, languageCode)
		exercises = append(exercises, exercise)
	}

	// Insert exercises with audio URLs
	for _, ex := range exercises {
		choicesJSON, _ := json.Marshal(ex.Choices)
		hintsJSON, _ := json.Marshal(ex.Hints)

		_, err := i.db.ExecContext(ctx, `
			INSERT INTO exercises (id, lesson_id, type, prompt, correct_answer, choices, audio_url, hints, difficulty)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.New().String(), lessonID, ex.Type, ex.Prompt, ex.CorrectAnswer,
			string(choicesJSON), ex.AudioURL, string(hintsJSON), ex.Difficulty)
		if err != nil {
			return fmt.Errorf("insert exercise: %w", err)
		}
	}

	return nil
}

// getExerciseType returns an exercise type based on index and level
// Duolingo-style progression:
// - Level 1: Recognition only (multiple_choice, match_pairs) - NO typing
// - Level 2: Add word_bank (tap words) - still no free typing
// - Level 3: Add fill_blank with choices
// - Level 4: Add translation with choices, listening with hints
// - Level 5: Free-form translation and listening (typing required)
func getExerciseType(index, level int) string {
	// Level 1: EASIEST - Only recognition, no production
	// Just identify meanings - perfect for absolute beginners
	level1Types := []string{
		"multiple_choice",
		"multiple_choice",
		"match_pairs",
		"multiple_choice",
		"multiple_choice",
		"match_pairs",
		"multiple_choice",
		"multiple_choice",
		"match_pairs",
		"multiple_choice",
		"match_pairs",
		"multiple_choice",
		"multiple_choice",
		"match_pairs",
		"multiple_choice",
	}

	// Level 2: Add word_bank (tap words to build answer)
	// Still no free typing - user taps words in order
	level2Types := []string{
		"multiple_choice",
		"word_bank",
		"match_pairs",
		"multiple_choice",
		"word_bank",
		"multiple_choice",
		"match_pairs",
		"word_bank",
		"multiple_choice",
		"word_bank",
		"match_pairs",
		"multiple_choice",
		"word_bank",
		"match_pairs",
		"multiple_choice",
	}

	// Level 3: Add fill_blank (with choices)
	// User selects from options, not free typing
	level3Types := []string{
		"multiple_choice",
		"word_bank",
		"fill_blank",
		"match_pairs",
		"multiple_choice",
		"fill_blank",
		"word_bank",
		"multiple_choice",
		"fill_blank",
		"match_pairs",
		"word_bank",
		"fill_blank",
		"multiple_choice",
		"word_bank",
		"fill_blank",
	}

	// Level 4: Add translation with choices, introduce listening
	// Translation still has choices, listening has strong hints
	level4Types := []string{
		"translation_choice", // Multiple choice translation
		"word_bank",
		"fill_blank",
		"match_pairs",
		"listening_choice", // Listening with choices
		"translation_choice",
		"word_bank",
		"fill_blank",
		"listening_choice",
		"translation_choice",
		"match_pairs",
		"word_bank",
		"fill_blank",
		"listening_choice",
		"translation_choice",
	}

	// Level 5: HARDEST - Free-form typing required
	// Full translation and listening exercises
	level5Types := []string{
		"translation",
		"listening",
		"fill_blank",
		"word_bank",
		"translation",
		"listening",
		"translation",
		"fill_blank",
		"listening",
		"translation",
		"word_bank",
		"listening",
		"translation",
		"fill_blank",
		"translation",
	}

	var types []string
	switch level {
	case 1:
		types = level1Types
	case 2:
		types = level2Types
	case 3:
		types = level3Types
	case 4:
		types = level4Types
	default:
		types = level5Types
	}

	return types[index%len(types)]
}

// generateExercise creates an exercise from vocabulary with audio support
func generateExercise(vocab VocabularyEntry, allVocab []VocabularyEntry, exType string, level int, languageCode string) store.Exercise {
	ex := store.Exercise{
		ID:         uuid.New(),
		Type:       exType,
		Difficulty: level,
	}

	translation := "unknown"
	if len(vocab.Translations) > 0 {
		translation = vocab.Translations[0]
	}

	// Generate audio URL for exercises that need it
	audioURL := vocab.AudioURL
	if audioURL == "" && LanguageHasAudio(languageCode) {
		audioURL = GenerateAudioURL(vocab.Word, languageCode)
	}

	switch exType {
	case "translation":
		// Free-form translation (typing required) - Level 5
		if rand.Intn(2) == 0 {
			ex.Prompt = fmt.Sprintf("Translate: %s", vocab.Word)
			ex.CorrectAnswer = translation
			ex.AudioURL = audioURL
		} else {
			ex.Prompt = fmt.Sprintf("Translate to the target language: %s", translation)
			ex.CorrectAnswer = vocab.Word
		}
		if vocab.Romanization != "" {
			ex.Hints = []string{fmt.Sprintf("Romanization: %s", vocab.Romanization)}
		}

	case "translation_choice":
		// Multiple choice translation (no typing) - Level 4
		ex.Type = "multiple_choice" // Use multiple_choice type for frontend
		if rand.Intn(2) == 0 {
			ex.Prompt = fmt.Sprintf("What is '%s' in English?", vocab.Word)
			ex.CorrectAnswer = translation
			ex.Choices = generateDistractors(translation, allVocab, 4)
			ex.AudioURL = audioURL
		} else {
			ex.Prompt = fmt.Sprintf("How do you say '%s'?", translation)
			ex.CorrectAnswer = vocab.Word
			ex.Choices = generateDistractorsWords(vocab.Word, allVocab, 4)
		}

	case "multiple_choice":
		ex.Prompt = fmt.Sprintf("What does '%s' mean?", vocab.Word)
		ex.CorrectAnswer = translation
		ex.Choices = generateDistractors(translation, allVocab, 4)
		ex.AudioURL = audioURL

	case "word_bank":
		ex.Prompt = translation
		ex.CorrectAnswer = vocab.Word
		ex.Choices = generateWordBankChoices(vocab.Word, allVocab)
		if vocab.Romanization != "" {
			ex.Hints = []string{vocab.Romanization}
		}

	case "fill_blank":
		ex.Prompt = fmt.Sprintf("___ means '%s'", translation)
		ex.CorrectAnswer = vocab.Word
		ex.Choices = generateDistractorsWords(vocab.Word, allVocab, 4)
		if vocab.Romanization != "" {
			ex.Hints = []string{vocab.Romanization}
		}

	case "listening":
		// Free-form listening (typing required) - Level 5
		ex.Prompt = "Type what you hear"
		ex.CorrectAnswer = vocab.Word
		ex.AudioURL = audioURL
		if vocab.Romanization != "" {
			ex.Hints = []string{fmt.Sprintf("Hint: %s", vocab.Romanization)}
		}

	case "listening_choice":
		// Multiple choice listening (no typing) - Level 4
		ex.Type = "multiple_choice" // Use multiple_choice type for frontend
		ex.Prompt = "What did you hear?"
		ex.CorrectAnswer = vocab.Word
		ex.Choices = generateDistractorsWords(vocab.Word, allVocab, 4)
		ex.AudioURL = audioURL
		if vocab.Romanization != "" {
			ex.Hints = []string{fmt.Sprintf("Romanization: %s", vocab.Romanization)}
		}

	case "match_pairs":
		ex.Prompt = "Match the words with their meanings"
		ex.CorrectAnswer = vocab.Word
		ex.Choices = generateMatchPairs(vocab, allVocab, 4)
	}

	return ex
}

// generateWordBankChoices creates word choices for word bank exercises
func generateWordBankChoices(correct string, allVocab []VocabularyEntry) []string {
	choices := []string{correct}

	// Add some distractor words
	for _, v := range allVocab {
		if v.Word != correct && len(choices) < 6 {
			choices = append(choices, v.Word)
		}
	}

	// Shuffle
	rand.Shuffle(len(choices), func(i, j int) {
		choices[i], choices[j] = choices[j], choices[i]
	})

	return choices
}

// generateDistractorsWords creates wrong word choices
func generateDistractorsWords(correct string, allVocab []VocabularyEntry, count int) []string {
	choices := []string{correct}

	// Collect unique words
	for _, v := range allVocab {
		if v.Word != correct && len(choices) < count {
			choices = append(choices, v.Word)
		}
	}

	// Shuffle
	rand.Shuffle(len(choices), func(i, j int) {
		choices[i], choices[j] = choices[j], choices[i]
	})

	return choices
}

// generateMatchPairs creates pairs for matching exercises
func generateMatchPairs(vocab VocabularyEntry, allVocab []VocabularyEntry, pairCount int) []string {
	// Format: word1|translation1,word2|translation2,...
	pairs := make([]string, 0, pairCount)

	// Add the correct pair
	if len(vocab.Translations) > 0 {
		pairs = append(pairs, fmt.Sprintf("%s|%s", vocab.Word, vocab.Translations[0]))
	}

	// Add distractor pairs
	for _, v := range allVocab {
		if v.Word != vocab.Word && len(v.Translations) > 0 && len(pairs) < pairCount {
			pairs = append(pairs, fmt.Sprintf("%s|%s", v.Word, v.Translations[0]))
		}
	}

	// Shuffle pairs
	rand.Shuffle(len(pairs), func(i, j int) {
		pairs[i], pairs[j] = pairs[j], pairs[i]
	})

	return pairs
}

// generateDistractors creates wrong answer choices
func generateDistractors(correct string, allVocab []VocabularyEntry, count int) []string {
	choices := []string{correct}

	// Collect all unique translations
	uniqueTranslations := make(map[string]bool)
	uniqueTranslations[correct] = true

	for _, v := range allVocab {
		if len(v.Translations) > 0 && v.Translations[0] != "" && v.Translations[0] != correct {
			uniqueTranslations[v.Translations[0]] = true
		}
	}

	// Convert to slice
	available := make([]string, 0, len(uniqueTranslations))
	for t := range uniqueTranslations {
		if t != correct {
			available = append(available, t)
		}
	}

	// Shuffle available distractors
	rand.Shuffle(len(available), func(i, j int) {
		available[i], available[j] = available[j], available[i]
	})

	// Add distractors up to count
	for i := 0; i < len(available) && len(choices) < count; i++ {
		choices = append(choices, available[i])
	}

	// Shuffle final choices
	rand.Shuffle(len(choices), func(i, j int) {
		choices[i], choices[j] = choices[j], choices[i]
	})

	return choices
}

// getIconForSkill returns an icon name based on skill name
func getIconForSkill(name string) string {
	name = strings.ToLower(name)

	iconMap := map[string]string{
		"basics":       "book",
		"greetings":    "hand-wave",
		"introduction": "user",
		"family":       "users",
		"food":         "utensils",
		"restaurant":   "utensils",
		"travel":       "plane",
		"transport":    "car",
		"shopping":     "shopping-cart",
		"clothes":      "shirt",
		"numbers":      "calculator",
		"colors":       "palette",
		"animals":      "paw",
		"nature":       "tree",
		"weather":      "cloud",
		"time":         "clock",
		"work":         "briefcase",
		"school":       "graduation-cap",
		"home":         "home",
		"health":       "heart",
		"sports":       "futbol",
		"music":        "music",
		"directions":   "compass",
	}

	for keyword, icon := range iconMap {
		if strings.Contains(name, keyword) {
			return icon
		}
	}

	return "book"
}

// ImportAll imports all parsed language pairs
func (i *Importer) ImportAll(ctx context.Context, pairs []LanguagePair) error {
	total := len(pairs)
	for idx, pair := range pairs {
		if i.progress != nil {
			i.progress(idx+1, total, fmt.Sprintf("Importing %s", pair))
		}

		if err := i.ImportPair(ctx, pair); err != nil {
			fmt.Printf("Warning: failed to import %s: %v\n", pair, err)
			continue
		}
	}
	return nil
}

// GetCourseID returns the course ID for a language pair
func (i *Importer) GetCourseID(ctx context.Context, pair LanguagePair) (string, error) {
	var courseID string
	err := i.db.QueryRowContext(ctx, `
		SELECT id FROM courses
		WHERE from_language_id = ? AND learning_language_id = ?
	`, pair.From, pair.To).Scan(&courseID)
	if err != nil {
		return "", fmt.Errorf("get course ID: %w", err)
	}
	return courseID, nil
}

// GetCourseStats returns statistics for a course
func (i *Importer) GetCourseStats(ctx context.Context, pair LanguagePair) (map[string]int, error) {
	stats := make(map[string]int)

	// Get course ID
	var courseID string
	err := i.db.QueryRowContext(ctx, `
		SELECT id FROM courses
		WHERE from_language_id = ? AND learning_language_id = ?
	`, pair.From, pair.To).Scan(&courseID)
	if err != nil {
		return nil, fmt.Errorf("get course: %w", err)
	}

	var count int

	// Count units
	err = i.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM units WHERE course_id = ?", courseID).Scan(&count)
	if err != nil {
		return nil, err
	}
	stats["units"] = count

	// Count skills (need to join through units)
	err = i.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM skills s
		JOIN units u ON s.unit_id = u.id
		WHERE u.course_id = ?
	`, courseID).Scan(&count)
	if err != nil {
		return nil, err
	}
	stats["skills"] = count

	// Count lessons
	err = i.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM lessons l
		JOIN skills s ON l.skill_id = s.id
		JOIN units u ON s.unit_id = u.id
		WHERE u.course_id = ?
	`, courseID).Scan(&count)
	if err != nil {
		return nil, err
	}
	stats["lessons"] = count

	// Count exercises
	err = i.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM exercises e
		JOIN lessons l ON e.lesson_id = l.id
		JOIN skills s ON l.skill_id = s.id
		JOIN units u ON s.unit_id = u.id
		WHERE u.course_id = ?
	`, courseID).Scan(&count)
	if err != nil {
		return nil, err
	}
	stats["exercises"] = count

	// Count lexemes
	err = i.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM lexemes WHERE course_id = ?", courseID).Scan(&count)
	if err != nil {
		return nil, err
	}
	stats["lexemes"] = count

	return stats, nil
}

// ImportStory imports a single story into the database
func (i *Importer) ImportStory(ctx context.Context, courseID string, story *StoryData, setID, setPosition int) error {
	storyID := uuid.New().String()

	// Determine CEFR level difficulty
	difficulty := 1
	switch {
	case strings.HasPrefix(story.CEFRLevel, "A1"):
		difficulty = 1
	case strings.HasPrefix(story.CEFRLevel, "A2"):
		difficulty = 2
	case strings.HasPrefix(story.CEFRLevel, "B1"):
		difficulty = 3
	case strings.HasPrefix(story.CEFRLevel, "B2"):
		difficulty = 4
	case strings.HasPrefix(story.CEFRLevel, "C"):
		difficulty = 5
	}

	// Calculate XP based on difficulty and element count
	xpReward := 10 + (difficulty * 5) + (len(story.Elements) / 2)

	// Estimate duration (3 seconds per element)
	duration := len(story.Elements) * 3

	// Insert story
	_, err := i.db.ExecContext(ctx, `
		INSERT INTO stories (id, course_id, external_id, title, title_translation, illustration_url,
			set_id, set_position, difficulty, cefr_level, duration_seconds, xp_reward, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`, storyID, courseID, story.ExternalID, story.Title, story.TitleTranslation, story.IllustrationURL,
		setID, setPosition, difficulty, story.CEFRLevel, duration, xpReward)
	if err != nil {
		return fmt.Errorf("insert story: %w", err)
	}

	// Insert characters
	characterIDMap := make(map[string]string)
	for pos, char := range story.Characters {
		charID := uuid.New().String()
		characterIDMap[char.Name] = charID

		_, err := i.db.ExecContext(ctx, `
			INSERT INTO story_characters (id, story_id, name, display_name, avatar_url, position)
			VALUES (?, ?, ?, ?, ?, ?)
		`, charID, storyID, char.Name, char.DisplayName, char.AvatarURL, pos)
		if err != nil {
			return fmt.Errorf("insert character %s: %w", char.Name, err)
		}
	}

	// Insert elements
	for _, elem := range story.Elements {
		elemID := uuid.New().String()

		var speakerID *string
		if elem.Speaker != "" {
			if id, ok := characterIDMap[elem.Speaker]; ok {
				speakerID = &id
			}
		}

		// Serialize challenge data if present
		var challengeDataJSON *string
		if elem.Challenge != nil {
			data, err := json.Marshal(elem.Challenge)
			if err == nil {
				s := string(data)
				challengeDataJSON = &s
			}
		}

		_, err := i.db.ExecContext(ctx, `
			INSERT INTO story_elements (id, story_id, position, element_type, speaker_id,
				text, translation, audio_url, challenge_data)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, elemID, storyID, elem.Position, elem.ElementType, speakerID,
			elem.Text, elem.Translation, elem.AudioURL, challengeDataJSON)
		if err != nil {
			return fmt.Errorf("insert element %d: %w", elem.Position, err)
		}
	}

	return nil
}

// ImportStoriesForCourse imports all stories for a course
func (i *Importer) ImportStoriesForCourse(ctx context.Context, courseID string, stories []*StoryData) error {
	// Delete existing stories for this course
	_, err := i.db.ExecContext(ctx, `
		DELETE FROM story_elements WHERE story_id IN (SELECT id FROM stories WHERE course_id = ?)
	`, courseID)
	if err != nil {
		return fmt.Errorf("delete story elements: %w", err)
	}

	_, err = i.db.ExecContext(ctx, `
		DELETE FROM story_characters WHERE story_id IN (SELECT id FROM stories WHERE course_id = ?)
	`, courseID)
	if err != nil {
		return fmt.Errorf("delete story characters: %w", err)
	}

	_, err = i.db.ExecContext(ctx, "DELETE FROM stories WHERE course_id = ?", courseID)
	if err != nil {
		return fmt.Errorf("delete stories: %w", err)
	}

	// Group stories by set (CEFR level)
	setMap := make(map[string]int)
	setCounter := 1

	for idx, story := range stories {
		setKey := story.CEFRLevel
		if setKey == "" {
			setKey = "Unknown"
		}

		setID, exists := setMap[setKey]
		if !exists {
			setID = setCounter
			setMap[setKey] = setID
			setCounter++
		}

		// Position within the set
		setPosition := idx + 1

		if err := i.ImportStory(ctx, courseID, story, setID, setPosition); err != nil {
			return fmt.Errorf("import story %s: %w", story.ExternalID, err)
		}
	}

	return nil
}
