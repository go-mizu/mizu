package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// Store implements store.Store using SQLite
type Store struct {
	db           *sql.DB
	users        *UserStore
	courses      *CourseStore
	progress     *ProgressStore
	gamification *GamificationStore
	social       *SocialStore
	achievements *AchievementStore
	stories      *StoryStore
}

// New creates a new SQLite store
func New(ctx context.Context, dbPath string) (*Store, error) {
	// For in-memory databases, use shared cache to allow multiple connections
	// to access the same database. WAL mode is not supported for in-memory.
	var dsn string
	if dbPath == ":memory:" {
		dsn = "file::memory:?cache=shared&_foreign_keys=on"
	} else {
		dsn = dbPath + "?_foreign_keys=on&_journal_mode=WAL"
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	s := &Store{db: db}
	s.users = &UserStore{db: db}
	s.courses = &CourseStore{db: db}
	s.progress = &ProgressStore{db: db}
	s.gamification = &GamificationStore{db: db}
	s.social = &SocialStore{db: db}
	s.achievements = &AchievementStore{db: db}
	s.stories = &StoryStore{db: db}

	return s, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection
func (s *Store) DB() *sql.DB {
	return s.db
}

// CreateExtensions is a no-op for SQLite
func (s *Store) CreateExtensions(ctx context.Context) error {
	return nil
}

// Ensure creates all database tables
func (s *Store) Ensure(ctx context.Context) error {
	statements := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			username TEXT UNIQUE NOT NULL,
			display_name TEXT,
			avatar_url TEXT,
			bio TEXT,
			encrypted_password TEXT NOT NULL,
			xp_total INTEGER DEFAULT 0,
			gems INTEGER DEFAULT 500,
			hearts INTEGER DEFAULT 5,
			hearts_updated_at DATETIME,
			streak_days INTEGER DEFAULT 0,
			streak_updated_at DATE,
			streak_freeze_count INTEGER DEFAULT 0,
			is_premium INTEGER DEFAULT 0,
			premium_expires_at DATETIME,
			daily_goal_minutes INTEGER DEFAULT 10,
			active_course_id TEXT REFERENCES courses(id),
			native_language_id TEXT DEFAULT 'en',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_active_at DATETIME
		)`,
		// Languages table
		`CREATE TABLE IF NOT EXISTS languages (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			native_name TEXT,
			flag_emoji TEXT,
			rtl INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1
		)`,
		// Courses table
		`CREATE TABLE IF NOT EXISTS courses (
			id TEXT PRIMARY KEY,
			from_language_id TEXT REFERENCES languages(id),
			learning_language_id TEXT REFERENCES languages(id),
			title TEXT,
			description TEXT,
			total_units INTEGER DEFAULT 0,
			cefr_level TEXT,
			enabled INTEGER DEFAULT 1
		)`,
		// User courses (enrollment)
		`CREATE TABLE IF NOT EXISTS user_courses (
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			course_id TEXT REFERENCES courses(id) ON DELETE CASCADE,
			current_unit_id TEXT,
			current_lesson_id TEXT,
			xp_earned INTEGER DEFAULT 0,
			crowns_earned INTEGER DEFAULT 0,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_practiced_at DATETIME,
			PRIMARY KEY (user_id, course_id)
		)`,
		// Units table
		`CREATE TABLE IF NOT EXISTS units (
			id TEXT PRIMARY KEY,
			course_id TEXT REFERENCES courses(id) ON DELETE CASCADE,
			position INTEGER NOT NULL,
			title TEXT,
			description TEXT,
			guidebook_content TEXT,
			icon_url TEXT
		)`,
		// Skills table
		`CREATE TABLE IF NOT EXISTS skills (
			id TEXT PRIMARY KEY,
			unit_id TEXT REFERENCES units(id) ON DELETE CASCADE,
			position INTEGER NOT NULL,
			name TEXT,
			icon_name TEXT,
			levels INTEGER DEFAULT 5,
			lexemes_count INTEGER DEFAULT 0
		)`,
		// Lessons table
		`CREATE TABLE IF NOT EXISTS lessons (
			id TEXT PRIMARY KEY,
			skill_id TEXT REFERENCES skills(id) ON DELETE CASCADE,
			level INTEGER NOT NULL,
			position INTEGER NOT NULL,
			exercise_count INTEGER DEFAULT 15
		)`,
		// Exercises table
		`CREATE TABLE IF NOT EXISTS exercises (
			id TEXT PRIMARY KEY,
			lesson_id TEXT REFERENCES lessons(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			prompt TEXT,
			correct_answer TEXT,
			choices TEXT,
			audio_url TEXT,
			image_url TEXT,
			hints TEXT,
			difficulty INTEGER DEFAULT 1
		)`,
		// Lexemes (vocabulary)
		`CREATE TABLE IF NOT EXISTS lexemes (
			id TEXT PRIMARY KEY,
			course_id TEXT REFERENCES courses(id) ON DELETE CASCADE,
			word TEXT NOT NULL,
			translation TEXT,
			pos TEXT,
			gender TEXT,
			audio_url TEXT,
			image_url TEXT,
			example_sentence TEXT,
			example_translation TEXT
		)`,
		// User skill progress
		`CREATE TABLE IF NOT EXISTS user_skills (
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			skill_id TEXT REFERENCES skills(id) ON DELETE CASCADE,
			crown_level INTEGER DEFAULT 0,
			is_legendary INTEGER DEFAULT 0,
			strength REAL DEFAULT 1.0,
			last_practiced_at DATETIME,
			next_review_at DATETIME,
			PRIMARY KEY (user_id, skill_id)
		)`,
		// User lexeme progress (spaced repetition)
		`CREATE TABLE IF NOT EXISTS user_lexemes (
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			lexeme_id TEXT REFERENCES lexemes(id) ON DELETE CASCADE,
			strength REAL DEFAULT 0,
			correct_count INTEGER DEFAULT 0,
			incorrect_count INTEGER DEFAULT 0,
			last_practiced_at DATETIME,
			next_review_at DATETIME,
			interval_days INTEGER DEFAULT 1,
			ease_factor REAL DEFAULT 2.5,
			PRIMARY KEY (user_id, lexeme_id)
		)`,
		// User mistakes
		`CREATE TABLE IF NOT EXISTS user_mistakes (
			id TEXT PRIMARY KEY,
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			exercise_id TEXT REFERENCES exercises(id) ON DELETE CASCADE,
			lexeme_id TEXT REFERENCES lexemes(id) ON DELETE SET NULL,
			user_answer TEXT,
			correct_answer TEXT,
			mistake_type TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Lesson sessions
		`CREATE TABLE IF NOT EXISTS lesson_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			lesson_id TEXT REFERENCES lessons(id) ON DELETE CASCADE,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			xp_earned INTEGER DEFAULT 0,
			mistakes_count INTEGER DEFAULT 0,
			hearts_lost INTEGER DEFAULT 0,
			is_perfect INTEGER DEFAULT 0
		)`,
		// XP events
		`CREATE TABLE IF NOT EXISTS xp_events (
			id TEXT PRIMARY KEY,
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			amount INTEGER NOT NULL,
			source TEXT,
			source_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Streak history
		`CREATE TABLE IF NOT EXISTS streak_history (
			id TEXT PRIMARY KEY,
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			date DATE NOT NULL,
			xp_earned INTEGER DEFAULT 0,
			lessons_completed INTEGER DEFAULT 0,
			time_spent_seconds INTEGER DEFAULT 0,
			freeze_used INTEGER DEFAULT 0,
			UNIQUE(user_id, date)
		)`,
		// Achievements
		`CREATE TABLE IF NOT EXISTS achievements (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			category TEXT,
			icon_url TEXT,
			max_level INTEGER DEFAULT 10,
			thresholds TEXT
		)`,
		// User achievements
		`CREATE TABLE IF NOT EXISTS user_achievements (
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			achievement_id TEXT REFERENCES achievements(id) ON DELETE CASCADE,
			level INTEGER DEFAULT 0,
			progress INTEGER DEFAULT 0,
			unlocked_at DATETIME,
			PRIMARY KEY (user_id, achievement_id)
		)`,
		// Leagues
		`CREATE TABLE IF NOT EXISTS leagues (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			icon_url TEXT,
			min_xp_to_promote INTEGER,
			demotion_zone_size INTEGER DEFAULT 5
		)`,
		// League seasons
		`CREATE TABLE IF NOT EXISTS league_seasons (
			id TEXT PRIMARY KEY,
			league_id INTEGER REFERENCES leagues(id),
			week_start DATE NOT NULL,
			week_end DATE NOT NULL
		)`,
		// User leagues
		`CREATE TABLE IF NOT EXISTS user_leagues (
			id TEXT PRIMARY KEY,
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			season_id TEXT REFERENCES league_seasons(id) ON DELETE CASCADE,
			xp_earned INTEGER DEFAULT 0,
			rank INTEGER,
			promoted INTEGER,
			demoted INTEGER,
			UNIQUE(user_id, season_id)
		)`,
		// Friends/follows
		`CREATE TABLE IF NOT EXISTS follows (
			follower_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			following_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (follower_id, following_id)
		)`,
		// Friend streaks
		`CREATE TABLE IF NOT EXISTS friend_streaks (
			id TEXT PRIMARY KEY,
			user1_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			user2_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			streak_days INTEGER DEFAULT 0,
			started_at DATE,
			last_both_active DATE
		)`,
		// Friend quests
		`CREATE TABLE IF NOT EXISTS friend_quests (
			id TEXT PRIMARY KEY,
			user1_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			user2_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			quest_type TEXT NOT NULL,
			target_value INTEGER NOT NULL,
			user1_progress INTEGER DEFAULT 0,
			user2_progress INTEGER DEFAULT 0,
			starts_at DATETIME NOT NULL,
			ends_at DATETIME NOT NULL,
			completed INTEGER DEFAULT 0,
			rewards_claimed INTEGER DEFAULT 0
		)`,
		// Story sets (collections)
		`CREATE TABLE IF NOT EXISTS story_sets (
			id INTEGER PRIMARY KEY,
			course_id TEXT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			description TEXT,
			position INTEGER NOT NULL,
			unlock_requirement TEXT,
			icon_url TEXT
		)`,
		// Stories
		`CREATE TABLE IF NOT EXISTS stories (
			id TEXT PRIMARY KEY,
			course_id TEXT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
			external_id TEXT,
			title TEXT NOT NULL,
			title_translation TEXT,
			illustration_url TEXT,
			set_id INTEGER NOT NULL DEFAULT 1,
			set_position INTEGER DEFAULT 0,
			difficulty INTEGER DEFAULT 1,
			cefr_level TEXT,
			duration_seconds INTEGER DEFAULT 180,
			xp_reward INTEGER DEFAULT 14,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(course_id, external_id)
		)`,
		// Story characters
		`CREATE TABLE IF NOT EXISTS story_characters (
			id TEXT PRIMARY KEY,
			story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			display_name TEXT,
			avatar_url TEXT,
			voice_id TEXT,
			position INTEGER DEFAULT 0,
			UNIQUE(story_id, name)
		)`,
		// Story elements (lines, challenges)
		`CREATE TABLE IF NOT EXISTS story_elements (
			id TEXT PRIMARY KEY,
			story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
			position INTEGER NOT NULL,
			element_type TEXT NOT NULL,
			speaker_id TEXT REFERENCES story_characters(id),
			text TEXT,
			translation TEXT,
			audio_url TEXT,
			audio_timing TEXT,
			challenge_data TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// User stories
		`CREATE TABLE IF NOT EXISTS user_stories (
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
			started_at DATETIME,
			completed_at DATETIME,
			completed INTEGER DEFAULT 0,
			xp_earned INTEGER DEFAULT 0,
			mistakes_count INTEGER DEFAULT 0,
			listen_mode_completed INTEGER DEFAULT 0,
			PRIMARY KEY (user_id, story_id)
		)`,
		// User story element progress
		`CREATE TABLE IF NOT EXISTS user_story_progress (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
			element_id TEXT NOT NULL REFERENCES story_elements(id) ON DELETE CASCADE,
			completed INTEGER DEFAULT 0,
			correct INTEGER,
			attempts INTEGER DEFAULT 0,
			completed_at DATETIME,
			UNIQUE(user_id, story_id, element_id)
		)`,
		// Notifications
		`CREATE TABLE IF NOT EXISTS notifications (
			id TEXT PRIMARY KEY,
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			title TEXT,
			body TEXT,
			data TEXT,
			read INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Purchases
		`CREATE TABLE IF NOT EXISTS purchases (
			id TEXT PRIMARY KEY,
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			item_type TEXT NOT NULL,
			item_id TEXT,
			gems_spent INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range statements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("execute schema statement: %w", err)
		}
	}

	// Run migrations for existing databases (add columns that might be missing)
	// This must happen BEFORE index creation so columns exist
	if err := s.runMigrations(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// Create indexes after migrations to ensure all columns exist
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_user_courses_user ON user_courses(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_units_course ON units(course_id)`,
		`CREATE INDEX IF NOT EXISTS idx_skills_unit ON skills(unit_id)`,
		`CREATE INDEX IF NOT EXISTS idx_lessons_skill ON lessons(skill_id)`,
		`CREATE INDEX IF NOT EXISTS idx_exercises_lesson ON exercises(lesson_id)`,
		`CREATE INDEX IF NOT EXISTS idx_xp_events_user ON xp_events(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_streak_history_user ON streak_history(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_leagues_season ON user_leagues(season_id)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_stories_course ON stories(course_id)`,
		`CREATE INDEX IF NOT EXISTS idx_stories_set ON stories(set_id)`,
		`CREATE INDEX IF NOT EXISTS idx_story_elements_story ON story_elements(story_id)`,
		`CREATE INDEX IF NOT EXISTS idx_story_elements_position ON story_elements(story_id, position)`,
		`CREATE INDEX IF NOT EXISTS idx_story_characters_story ON story_characters(story_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_stories_user ON user_stories(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_stories_story ON user_stories(story_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_story_progress_user ON user_story_progress(user_id, story_id)`,
	}

	for _, stmt := range indexes {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	return nil
}

// runMigrations adds missing columns to existing tables
func (s *Store) runMigrations(ctx context.Context) error {
	// Check if active_course_id column exists in users table
	if !s.columnExists(ctx, "users", "active_course_id") {
		_, err := s.db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN active_course_id TEXT REFERENCES courses(id)`)
		if err != nil {
			return fmt.Errorf("add active_course_id column: %w", err)
		}
	}

	// Check if native_language_id column exists in users table
	if !s.columnExists(ctx, "users", "native_language_id") {
		_, err := s.db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN native_language_id TEXT DEFAULT 'en'`)
		if err != nil {
			return fmt.Errorf("add native_language_id column: %w", err)
		}
	}

	// Check if set_id column exists in stories table
	if s.tableExists(ctx, "stories") && !s.columnExists(ctx, "stories", "set_id") {
		_, err := s.db.ExecContext(ctx, `ALTER TABLE stories ADD COLUMN set_id INTEGER NOT NULL DEFAULT 1`)
		if err != nil {
			return fmt.Errorf("add set_id column to stories: %w", err)
		}
	}

	// Check if set_position column exists in stories table
	if s.tableExists(ctx, "stories") && !s.columnExists(ctx, "stories", "set_position") {
		_, err := s.db.ExecContext(ctx, `ALTER TABLE stories ADD COLUMN set_position INTEGER DEFAULT 0`)
		if err != nil {
			return fmt.Errorf("add set_position column to stories: %w", err)
		}
	}

	return nil
}

// tableExists checks if a table exists in the database
func (s *Store) tableExists(ctx context.Context, table string) bool {
	var name string
	err := s.db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
	return err == nil
}

// columnExists checks if a column exists in a table
func (s *Store) columnExists(ctx context.Context, table, column string) bool {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			continue
		}
		if name == column {
			return true
		}
	}
	return false
}

// Users returns the user store
func (s *Store) Users() store.UserStore {
	return s.users
}

// Courses returns the course store
func (s *Store) Courses() store.CourseStore {
	return s.courses
}

// Progress returns the progress store
func (s *Store) Progress() store.ProgressStore {
	return s.progress
}

// Gamification returns the gamification store
func (s *Store) Gamification() store.GamificationStore {
	return s.gamification
}

// Social returns the social store
func (s *Store) Social() store.SocialStore {
	return s.social
}

// Achievements returns the achievement store
func (s *Store) Achievements() store.AchievementStore {
	return s.achievements
}

// Stories returns the story store
func (s *Store) Stories() store.StoryStore {
	return s.stories
}

// ============================================================================
// Seeding functions
// ============================================================================

// SeedLanguages seeds the languages table with all supported languages
func (s *Store) SeedLanguages(ctx context.Context) error {
	// Comprehensive list of all supported languages
	languages := []struct {
		ID         string
		Name       string
		NativeName string
		FlagEmoji  string
		RTL        bool
	}{
		// Major languages
		{"en", "English", "English", "🇺🇸", false},
		{"es", "Spanish", "Español", "🇪🇸", false},
		{"fr", "French", "Français", "🇫🇷", false},
		{"de", "German", "Deutsch", "🇩🇪", false},
		{"it", "Italian", "Italiano", "🇮🇹", false},
		{"pt", "Portuguese", "Português", "🇧🇷", false},
		{"nl", "Dutch", "Nederlands", "🇳🇱", false},
		{"sv", "Swedish", "Svenska", "🇸🇪", false},
		{"nb", "Norwegian Bokmål", "Norsk Bokmål", "🇳🇴", false},
		{"da", "Danish", "Dansk", "🇩🇰", false},
		{"fi", "Finnish", "Suomi", "🇫🇮", false},
		{"ru", "Russian", "Русский", "🇷🇺", false},
		{"tr", "Turkish", "Türkçe", "🇹🇷", false},
		{"ar", "Arabic", "العربية", "🇸🇦", true},
		{"ja", "Japanese", "日本語", "🇯🇵", false},
		{"ko", "Korean", "한국어", "🇰🇷", false},
		{"zh", "Chinese", "中文", "🇨🇳", false},
		{"zs", "Chinese (Simplified)", "简体中文", "🇨🇳", false},
		{"zc", "Cantonese", "廣東話", "🇭🇰", false},
		{"hu", "Hungarian", "Magyar", "🇭🇺", false},
		{"ro", "Romanian", "Română", "🇷🇴", false},
		{"ga", "Irish", "Gaeilge", "🇮🇪", false},
		{"gd", "Scottish Gaelic", "Gàidhlig", "🏴󠁧󠁢󠁳󠁣󠁴󠁿", false},
		{"cy", "Welsh", "Cymraeg", "🏴󠁧󠁢󠁷󠁬󠁳󠁿", false},
		{"ca", "Catalan", "Català", "🏴󠁥󠁳󠁣󠁴󠁿", false},
		{"pl", "Polish", "Polski", "🇵🇱", false},
		{"uk", "Ukrainian", "Українська", "🇺🇦", false},
		{"cs", "Czech", "Čeština", "🇨🇿", false},
		{"el", "Greek", "Ελληνικά", "🇬🇷", false},
		{"he", "Hebrew", "עברית", "🇮🇱", true},
		{"hi", "Hindi", "हिन्दी", "🇮🇳", false},
		{"vi", "Vietnamese", "Tiếng Việt", "🇻🇳", false},
		{"id", "Indonesian", "Bahasa Indonesia", "🇮🇩", false},
		{"th", "Thai", "ภาษาไทย", "🇹🇭", false},
		// Constructed and special languages
		{"eo", "Esperanto", "Esperanto", "🟢", false},
		{"la", "Latin", "Latina", "🏛️", false},
		{"kl", "Klingon", "tlhIngan Hol", "🖖", false},
		{"hv", "High Valyrian", "High Valyrian", "🐉", false},
		// Indigenous and regional languages
		{"gn", "Guarani", "Avañe'ẽ", "🇵🇾", false},
		{"hw", "Hawaiian", "ʻŌlelo Hawaiʻi", "🌺", false},
		{"nv", "Navajo", "Diné bizaad", "🏜️", false},
		{"sw", "Swahili", "Kiswahili", "🇰🇪", false},
		{"zu", "Zulu", "isiZulu", "🇿🇦", false},
		{"yi", "Yiddish", "ייִדיש", "🕎", true},
		{"ht", "Haitian Creole", "Kreyòl ayisyen", "🇭🇹", false},
		{"dn", "Dutch", "Nederlands", "🇳🇱", false},
	}

	for _, lang := range languages {
		_, err := s.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO languages (id, name, native_name, flag_emoji, rtl, enabled)
			VALUES (?, ?, ?, ?, ?, 1)
		`, lang.ID, lang.Name, lang.NativeName, lang.FlagEmoji, boolToInt(lang.RTL))
		if err != nil {
			return fmt.Errorf("insert language %s: %w", lang.ID, err)
		}
	}

	return nil
}

// SeedCourses seeds courses with units, skills, lessons, and exercises
func (s *Store) SeedCourses(ctx context.Context) error {
	courses := []struct {
		FromLang    string
		LearnLang   string
		Title       string
		Description string
	}{
		{"en", "es", "Spanish for English Speakers", "Learn Spanish from scratch"},
		{"en", "fr", "French for English Speakers", "Learn French from scratch"},
		{"en", "de", "German for English Speakers", "Learn German from scratch"},
		{"en", "ja", "Japanese for English Speakers", "Learn Japanese from scratch"},
	}

	for _, c := range courses {
		courseID := uuid.New().String()
		_, err := s.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO courses (id, from_language_id, learning_language_id, title, description, total_units, cefr_level, enabled)
			VALUES (?, ?, ?, ?, ?, 10, 'A1', 1)
		`, courseID, c.FromLang, c.LearnLang, c.Title, c.Description)
		if err != nil {
			return fmt.Errorf("insert course %s: %w", c.Title, err)
		}

		if err := s.seedCourseContent(ctx, courseID, c.LearnLang); err != nil {
			return fmt.Errorf("seed course content for %s: %w", c.Title, err)
		}
	}

	return nil
}

func (s *Store) seedCourseContent(ctx context.Context, courseID, learnLang string) error {
	units := []struct {
		Title       string
		Description string
		Skills      []string
	}{
		{"Basics 1", "Learn basic greetings and introductions", []string{"Greetings", "Introduction", "Common Phrases"}},
		{"Basics 2", "Learn more fundamental vocabulary", []string{"Family", "Numbers", "Colors"}},
		{"Food", "Learn food vocabulary", []string{"Fruits", "Vegetables", "Drinks"}},
		{"Travel", "Learn travel-related words", []string{"Directions", "Transportation", "Hotels"}},
		{"Shopping", "Learn shopping vocabulary", []string{"Clothes", "Money", "Stores"}},
	}

	vocab := map[string][]struct {
		Word        string
		Translation string
	}{
		"Greetings": {
			{"Hola", "Hello"},
			{"Buenos días", "Good morning"},
			{"Buenas noches", "Good night"},
			{"Adiós", "Goodbye"},
			{"Gracias", "Thank you"},
		},
		"Family": {
			{"madre", "mother"},
			{"padre", "father"},
			{"hermano", "brother"},
			{"hermana", "sister"},
			{"hijo", "son"},
		},
		"Numbers": {
			{"uno", "one"},
			{"dos", "two"},
			{"tres", "three"},
			{"cuatro", "four"},
			{"cinco", "five"},
		},
	}

	for unitPos, unit := range units {
		unitID := uuid.New().String()
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO units (id, course_id, position, title, description, guidebook_content)
			VALUES (?, ?, ?, ?, ?, ?)
		`, unitID, courseID, unitPos+1, unit.Title, unit.Description, "Welcome to "+unit.Title)
		if err != nil {
			return fmt.Errorf("insert unit: %w", err)
		}

		for skillPos, skillName := range unit.Skills {
			skillID := uuid.New().String()
			_, err := s.db.ExecContext(ctx, `
				INSERT INTO skills (id, unit_id, position, name, icon_name, levels, lexemes_count)
				VALUES (?, ?, ?, ?, ?, 5, 5)
			`, skillID, unitID, skillPos+1, skillName, "book")
			if err != nil {
				return fmt.Errorf("insert skill: %w", err)
			}

			for level := 1; level <= 5; level++ {
				lessonID := uuid.New().String()
				_, err := s.db.ExecContext(ctx, `
					INSERT INTO lessons (id, skill_id, level, position, exercise_count)
					VALUES (?, ?, ?, ?, 15)
				`, lessonID, skillID, level, 1)
				if err != nil {
					return fmt.Errorf("insert lesson: %w", err)
				}

				if err := s.seedExercises(ctx, lessonID, skillName, vocab); err != nil {
					return fmt.Errorf("seed exercises: %w", err)
				}
			}
		}
	}

	return nil
}

func (s *Store) seedExercises(ctx context.Context, lessonID, skillName string, vocab map[string][]struct {
	Word        string
	Translation string
}) error {
	exerciseTypes := []string{
		"translation",
		"multiple_choice",
		"word_bank",
		"listening",
		"fill_blank",
		"match_pairs",
	}

	words := vocab[skillName]
	if len(words) == 0 {
		words = vocab["Greetings"]
	}

	for i := 0; i < 15; i++ {
		word := words[i%len(words)]
		exType := exerciseTypes[i%len(exerciseTypes)]

		var prompt, answer string
		var choices []string

		switch exType {
		case "translation":
			prompt = fmt.Sprintf("Translate: %s", word.Word)
			answer = word.Translation
		case "multiple_choice":
			prompt = fmt.Sprintf("What does '%s' mean?", word.Word)
			answer = word.Translation
			choices = []string{word.Translation, "incorrect1", "incorrect2", "incorrect3"}
		case "word_bank":
			prompt = fmt.Sprintf("Build: %s", word.Translation)
			answer = word.Word
		case "listening":
			prompt = "Type what you hear"
			answer = word.Word
		case "fill_blank":
			prompt = fmt.Sprintf("___ means %s", word.Translation)
			answer = word.Word
		case "match_pairs":
			prompt = "Match the words"
			answer = word.Word
		}

		choicesJSON, _ := json.Marshal(choices)
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO exercises (id, lesson_id, type, prompt, correct_answer, choices, difficulty)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, uuid.New().String(), lessonID, exType, prompt, answer, string(choicesJSON), 1)
		if err != nil {
			return fmt.Errorf("insert exercise: %w", err)
		}
	}

	return nil
}

// SeedAchievements seeds the achievements table
func (s *Store) SeedAchievements(ctx context.Context) error {
	achievements := []struct {
		ID          string
		Name        string
		Description string
		Category    string
		MaxLevel    int
		Thresholds  []int
	}{
		{"wildfire", "Wildfire", "Reach a streak", "streak", 10, []int{3, 7, 14, 30, 60, 90, 180, 365, 500, 1000}},
		{"xp_olympian", "XP Olympian", "Earn XP", "xp", 10, []int{100, 500, 1000, 2500, 5000, 10000, 15000, 20000, 25000, 30000}},
		{"scholar", "Scholar", "Learn words", "learning", 10, []int{50, 100, 250, 500, 750, 1000, 1500, 2000, 3000, 5000}},
		{"sage", "Sage", "Complete lessons", "learning", 10, []int{10, 25, 50, 100, 200, 350, 500, 750, 1000, 1500}},
		{"social_butterfly", "Social Butterfly", "Follow friends", "social", 5, []int{1, 3, 5, 10, 20}},
		{"winner", "Winner", "Win leagues", "league", 10, []int{1, 3, 5, 10, 25, 50, 75, 100, 150, 200}},
		{"champion", "Champion", "Reach Diamond", "league", 1, []int{1}},
		{"perfect", "Perfectionist", "Perfect lessons", "learning", 10, []int{1, 5, 10, 25, 50, 100, 200, 350, 500, 750}},
		{"early_bird", "Early Bird", "Practice before 7 AM", "special", 5, []int{1, 7, 30, 100, 365}},
		{"night_owl", "Night Owl", "Practice after 10 PM", "special", 5, []int{1, 7, 30, 100, 365}},
		{"photogenic", "Photogenic", "Add profile picture", "social", 1, []int{1}},
	}

	for _, a := range achievements {
		thresholdsJSON, _ := json.Marshal(a.Thresholds)
		_, err := s.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO achievements (id, name, description, category, max_level, thresholds)
			VALUES (?, ?, ?, ?, ?, ?)
		`, a.ID, a.Name, a.Description, a.Category, a.MaxLevel, string(thresholdsJSON))
		if err != nil {
			return fmt.Errorf("insert achievement %s: %w", a.ID, err)
		}
	}

	return nil
}

// SeedLeagues seeds the leagues table
func (s *Store) SeedLeagues(ctx context.Context) error {
	leagues := []struct {
		ID               int
		Name             string
		MinXPToPromote   int
		DemotionZoneSize int
	}{
		{1, "Bronze", 50, 5},
		{2, "Silver", 100, 5},
		{3, "Gold", 200, 5},
		{4, "Sapphire", 350, 5},
		{5, "Ruby", 500, 5},
		{6, "Emerald", 750, 5},
		{7, "Amethyst", 1000, 5},
		{8, "Pearl", 1500, 5},
		{9, "Obsidian", 2000, 5},
		{10, "Diamond", 0, 5},
	}

	for _, l := range leagues {
		_, err := s.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO leagues (id, name, min_xp_to_promote, demotion_zone_size)
			VALUES (?, ?, ?, ?)
		`, l.ID, l.Name, l.MinXPToPromote, l.DemotionZoneSize)
		if err != nil {
			return fmt.Errorf("insert league %s: %w", l.Name, err)
		}
	}

	return nil
}

// SeedUsers creates sample users
func (s *Store) SeedUsers(ctx context.Context) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	users := []struct {
		Email       string
		Username    string
		DisplayName string
		XP          int64
		Streak      int
		Gems        int
	}{
		{"demo@lingo.dev", "demo", "Demo User", 5000, 30, 1500},
		{"admin@lingo.dev", "admin", "Admin User", 25000, 100, 5000},
		{"learner@lingo.dev", "learner", "Active Learner", 12500, 60, 2500},
	}

	for _, u := range users {
		_, err := s.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO users (id, email, username, display_name, encrypted_password, xp_total, streak_days, gems, hearts, daily_goal_minutes)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 5, 10)
		`, uuid.New().String(), u.Email, u.Username, u.DisplayName, string(hashedPassword), u.XP, u.Streak, u.Gems)
		if err != nil {
			return fmt.Errorf("insert user %s: %w", u.Email, err)
		}
	}

	return nil
}

// Helper functions
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
