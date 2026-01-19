package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Store defines the interface for all data operations
type Store interface {
	// Lifecycle
	Close() error
	CreateExtensions(ctx context.Context) error
	Ensure(ctx context.Context) error

	// Seeding
	SeedLanguages(ctx context.Context) error
	SeedCourses(ctx context.Context) error
	SeedAchievements(ctx context.Context) error
	SeedLeagues(ctx context.Context) error
	SeedUsers(ctx context.Context) error

	// User operations
	Users() UserStore
	// Course operations
	Courses() CourseStore
	// Progress operations
	Progress() ProgressStore
	// Gamification operations
	Gamification() GamificationStore
	// Social operations
	Social() SocialStore
	// Achievements operations
	Achievements() AchievementStore
	// Stories operations
	Stories() StoryStore
}

// UserStore handles user operations
type UserStore interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, user *User) error
	UpdateXP(ctx context.Context, userID uuid.UUID, amount int) error
	UpdateStreak(ctx context.Context, userID uuid.UUID) error
	UpdateHearts(ctx context.Context, userID uuid.UUID, hearts int) error
	UpdateGems(ctx context.Context, userID uuid.UUID, gems int) error
	SetActiveCourse(ctx context.Context, userID, courseID uuid.UUID) error
}

// CourseStore handles course operations
type CourseStore interface {
	ListLanguages(ctx context.Context) ([]Language, error)
	ListCourses(ctx context.Context, fromLang string) ([]Course, error)
	GetCourse(ctx context.Context, id uuid.UUID) (*Course, error)
	GetCoursePath(ctx context.Context, courseID uuid.UUID) ([]Unit, error)
	GetUnit(ctx context.Context, id uuid.UUID) (*Unit, error)
	GetSkill(ctx context.Context, id uuid.UUID) (*Skill, error)
	GetLesson(ctx context.Context, id uuid.UUID) (*Lesson, error)
	GetLessonsBySkill(ctx context.Context, skillID uuid.UUID) ([]Lesson, error)
	GetExercises(ctx context.Context, lessonID uuid.UUID) ([]Exercise, error)
	GetLexemesByCourse(ctx context.Context, courseID uuid.UUID) ([]Lexeme, error)
	GetStories(ctx context.Context, courseID uuid.UUID) ([]Story, error)
	GetStory(ctx context.Context, id uuid.UUID) (*Story, error)
}

// ProgressStore handles user progress operations
type ProgressStore interface {
	EnrollCourse(ctx context.Context, userID, courseID uuid.UUID) error
	GetUserCourses(ctx context.Context, userID uuid.UUID) ([]UserCourse, error)
	GetUserCourse(ctx context.Context, userID, courseID uuid.UUID) (*UserCourse, error)
	UpdateUserCourse(ctx context.Context, uc *UserCourse) error
	GetUserSkill(ctx context.Context, userID, skillID uuid.UUID) (*UserSkill, error)
	UpdateUserSkill(ctx context.Context, us *UserSkill) error
	GetUserLexemes(ctx context.Context, userID uuid.UUID, limit int) ([]UserLexeme, error)
	UpdateUserLexeme(ctx context.Context, ul *UserLexeme) error
	RecordMistake(ctx context.Context, mistake *UserMistake) error
	GetUserMistakes(ctx context.Context, userID uuid.UUID, limit int) ([]UserMistake, error)
	StartLessonSession(ctx context.Context, session *LessonSession) error
	CompleteLessonSession(ctx context.Context, session *LessonSession) error
	RecordXPEvent(ctx context.Context, event *XPEvent) error
	GetXPHistory(ctx context.Context, userID uuid.UUID, days int) ([]XPEvent, error)
	RecordStreakDay(ctx context.Context, userID uuid.UUID, xp, lessons, seconds int) error
	GetStreakHistory(ctx context.Context, userID uuid.UUID, days int) ([]StreakDay, error)
}

// GamificationStore handles gamification operations
type GamificationStore interface {
	GetLeagues(ctx context.Context) ([]League, error)
	GetCurrentSeason(ctx context.Context, leagueID int) (*LeagueSeason, error)
	GetLeaderboard(ctx context.Context, seasonID uuid.UUID, limit int) ([]UserLeague, error)
	GetUserLeague(ctx context.Context, userID uuid.UUID) (*UserLeague, error)
	JoinLeague(ctx context.Context, userID uuid.UUID, seasonID uuid.UUID) error
	UpdateLeagueXP(ctx context.Context, userID, seasonID uuid.UUID, xp int) error
	ProcessWeeklyLeagues(ctx context.Context) error
}

// SocialStore handles social operations
type SocialStore interface {
	Follow(ctx context.Context, followerID, followingID uuid.UUID) error
	Unfollow(ctx context.Context, followerID, followingID uuid.UUID) error
	GetFollowers(ctx context.Context, userID uuid.UUID) ([]User, error)
	GetFollowing(ctx context.Context, userID uuid.UUID) ([]User, error)
	GetFriendLeaderboard(ctx context.Context, userID uuid.UUID) ([]User, error)
	GetFriendQuests(ctx context.Context, userID uuid.UUID) ([]FriendQuest, error)
	CreateFriendQuest(ctx context.Context, quest *FriendQuest) error
	UpdateFriendQuest(ctx context.Context, quest *FriendQuest) error
	GetFriendStreaks(ctx context.Context, userID uuid.UUID) ([]FriendStreak, error)
	UpdateFriendStreak(ctx context.Context, streak *FriendStreak) error
	CreateNotification(ctx context.Context, notif *Notification) error
	GetNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool) ([]Notification, error)
	MarkNotificationRead(ctx context.Context, id uuid.UUID) error
}

// AchievementStore handles achievement operations
type AchievementStore interface {
	GetAchievements(ctx context.Context) ([]Achievement, error)
	GetUserAchievements(ctx context.Context, userID uuid.UUID) ([]UserAchievement, error)
	UpdateUserAchievement(ctx context.Context, ua *UserAchievement) error
	CheckAndUnlock(ctx context.Context, userID uuid.UUID, achievementID string, progress int) (*UserAchievement, error)
}

// StoryStore handles story operations
type StoryStore interface {
	// Stories
	GetStorySets(ctx context.Context, courseID uuid.UUID) ([]StorySet, error)
	GetStories(ctx context.Context, courseID uuid.UUID) ([]Story, error)
	GetStoriesBySet(ctx context.Context, courseID uuid.UUID, setID int) ([]Story, error)
	GetStory(ctx context.Context, id uuid.UUID) (*Story, error)
	GetStoryElements(ctx context.Context, storyID uuid.UUID) ([]StoryElement, error)
	GetStoryCharacters(ctx context.Context, storyID uuid.UUID) ([]StoryCharacter, error)

	// User progress
	GetUserStory(ctx context.Context, userID, storyID uuid.UUID) (*UserStory, error)
	GetUserStories(ctx context.Context, userID, courseID uuid.UUID) ([]UserStory, error)
	StartStory(ctx context.Context, userID, storyID uuid.UUID) error
	CompleteStory(ctx context.Context, userID, storyID uuid.UUID, xp, mistakes int) error
	RecordElementProgress(ctx context.Context, progress *UserStoryProgress) error
	GetStoryProgress(ctx context.Context, userID, storyID uuid.UUID) ([]UserStoryProgress, error)

	// Seeding/Import
	CreateStory(ctx context.Context, story *Story) error
	CreateStoryCharacter(ctx context.Context, char *StoryCharacter) error
	CreateStoryElement(ctx context.Context, elem *StoryElement) error
	CreateStorySet(ctx context.Context, set *StorySet) error
	DeleteStoriesByCourse(ctx context.Context, courseID uuid.UUID) error
}

// ============================================================================
// Data Types
// ============================================================================

// User represents a user account
type User struct {
	ID                uuid.UUID  `json:"id"`
	Email             string     `json:"email"`
	Username          string     `json:"username"`
	DisplayName       string     `json:"display_name"`
	AvatarURL         string     `json:"avatar_url,omitempty"`
	Bio               string     `json:"bio,omitempty"`
	EncryptedPassword string     `json:"-"`
	XPTotal           int64      `json:"xp_total"`
	Gems              int        `json:"gems"`
	Hearts            int        `json:"hearts"`
	HeartsUpdatedAt   *time.Time `json:"hearts_updated_at,omitempty"`
	StreakDays        int        `json:"streak_days"`
	StreakUpdatedAt   *time.Time `json:"streak_updated_at,omitempty"`
	StreakFreezeCount int        `json:"streak_freeze_count"`
	IsPremium         bool       `json:"is_premium"`
	PremiumExpiresAt  *time.Time `json:"premium_expires_at,omitempty"`
	DailyGoalMinutes  int        `json:"daily_goal_minutes"`
	ActiveCourseID    *uuid.UUID `json:"active_course_id,omitempty"`
	NativeLanguageID  string     `json:"native_language_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	LastActiveAt      *time.Time `json:"last_active_at,omitempty"`
}

// Language represents a language
type Language struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	NativeName    string `json:"native_name"`
	FlagEmoji     string `json:"flag_emoji"`
	RTL           bool   `json:"rtl"`
	Enabled       bool   `json:"enabled"`
	LearnersCount int64  `json:"learners_count,omitempty"`
}

// Course represents a language course
type Course struct {
	ID                 uuid.UUID `json:"id"`
	FromLanguageID     string    `json:"from_language_id"`
	LearningLanguageID string    `json:"learning_language_id"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	TotalUnits         int       `json:"total_units"`
	CEFRLevel          string    `json:"cefr_level"`
	Enabled            bool      `json:"enabled"`
}

// UserCourse represents a user's enrollment in a course
type UserCourse struct {
	UserID           uuid.UUID  `json:"user_id"`
	CourseID         uuid.UUID  `json:"course_id"`
	CurrentUnitID    *uuid.UUID `json:"current_unit_id,omitempty"`
	CurrentLessonID  *uuid.UUID `json:"current_lesson_id,omitempty"`
	XPEarned         int64      `json:"xp_earned"`
	CrownsEarned     int        `json:"crowns_earned"`
	StartedAt        time.Time  `json:"started_at"`
	LastPracticedAt  *time.Time `json:"last_practiced_at,omitempty"`
}

// Unit represents a unit in a course
type Unit struct {
	ID               uuid.UUID `json:"id"`
	CourseID         uuid.UUID `json:"course_id"`
	Position         int       `json:"position"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	GuidebookContent string    `json:"guidebook_content,omitempty"`
	IconURL          string    `json:"icon_url,omitempty"`
	Skills           []Skill   `json:"skills,omitempty"`
}

// Skill represents a skill within a unit
type Skill struct {
	ID           uuid.UUID `json:"id"`
	UnitID       uuid.UUID `json:"unit_id"`
	Position     int       `json:"position"`
	Name         string    `json:"name"`
	IconName     string    `json:"icon_name"`
	Levels       int       `json:"levels"`
	LexemesCount int       `json:"lexemes_count"`
	Lessons      []Lesson  `json:"lessons,omitempty"`
}

// Lesson represents a lesson within a skill
type Lesson struct {
	ID            uuid.UUID  `json:"id"`
	SkillID       uuid.UUID  `json:"skill_id"`
	Level         int        `json:"level"`
	Position      int        `json:"position"`
	ExerciseCount int        `json:"exercise_count"`
	Exercises     []Exercise `json:"exercises,omitempty"`
}

// Exercise represents an exercise in a lesson
type Exercise struct {
	ID            uuid.UUID `json:"id"`
	LessonID      uuid.UUID `json:"lesson_id"`
	Type          string    `json:"type"`
	Prompt        string    `json:"prompt"`
	CorrectAnswer string    `json:"correct_answer"`
	Choices       []string  `json:"choices,omitempty"`
	AudioURL      string    `json:"audio_url,omitempty"`
	ImageURL      string    `json:"image_url,omitempty"`
	Hints         []string  `json:"hints,omitempty"`
	Difficulty    int       `json:"difficulty"`
}

// Lexeme represents a vocabulary word
type Lexeme struct {
	ID                 uuid.UUID `json:"id"`
	CourseID           uuid.UUID `json:"course_id"`
	Word               string    `json:"word"`
	Translation        string    `json:"translation"`
	POS                string    `json:"pos"`
	Gender             string    `json:"gender,omitempty"`
	AudioURL           string    `json:"audio_url,omitempty"`
	ImageURL           string    `json:"image_url,omitempty"`
	ExampleSentence    string    `json:"example_sentence,omitempty"`
	ExampleTranslation string    `json:"example_translation,omitempty"`
}

// UserSkill represents a user's progress on a skill
type UserSkill struct {
	UserID          uuid.UUID  `json:"user_id"`
	SkillID         uuid.UUID  `json:"skill_id"`
	CrownLevel      int        `json:"crown_level"`
	IsLegendary     bool       `json:"is_legendary"`
	Strength        float64    `json:"strength"`
	LastPracticedAt *time.Time `json:"last_practiced_at,omitempty"`
	NextReviewAt    *time.Time `json:"next_review_at,omitempty"`
}

// UserLexeme represents a user's progress on a vocabulary word
type UserLexeme struct {
	UserID          uuid.UUID  `json:"user_id"`
	LexemeID        uuid.UUID  `json:"lexeme_id"`
	Strength        float64    `json:"strength"`
	CorrectCount    int        `json:"correct_count"`
	IncorrectCount  int        `json:"incorrect_count"`
	LastPracticedAt *time.Time `json:"last_practiced_at,omitempty"`
	NextReviewAt    *time.Time `json:"next_review_at,omitempty"`
	IntervalDays    int        `json:"interval_days"`
	EaseFactor      float64    `json:"ease_factor"`
}

// UserMistake represents a mistake made by a user
type UserMistake struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	ExerciseID    uuid.UUID `json:"exercise_id"`
	LexemeID      uuid.UUID `json:"lexeme_id,omitempty"`
	UserAnswer    string    `json:"user_answer"`
	CorrectAnswer string    `json:"correct_answer"`
	MistakeType   string    `json:"mistake_type"`
	CreatedAt     time.Time `json:"created_at"`
}

// LessonSession represents a lesson attempt
type LessonSession struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	LessonID      uuid.UUID  `json:"lesson_id"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	XPEarned      int        `json:"xp_earned"`
	MistakesCount int        `json:"mistakes_count"`
	HeartsLost    int        `json:"hearts_lost"`
	IsPerfect     bool       `json:"is_perfect"`
}

// XPEvent represents an XP earning event
type XPEvent struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Amount    int       `json:"amount"`
	Source    string    `json:"source"`
	SourceID  uuid.UUID `json:"source_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// StreakDay represents a day in the user's streak history
type StreakDay struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	Date             time.Time `json:"date"`
	XPEarned         int       `json:"xp_earned"`
	LessonsCompleted int       `json:"lessons_completed"`
	TimeSpentSeconds int       `json:"time_spent_seconds"`
	FreezeUsed       bool      `json:"freeze_used"`
}

// Achievement represents an achievement definition
type Achievement struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	IconURL     string `json:"icon_url,omitempty"`
	MaxLevel    int    `json:"max_level"`
	Thresholds  []int  `json:"thresholds"`
}

// UserAchievement represents a user's progress on an achievement
type UserAchievement struct {
	UserID        uuid.UUID  `json:"user_id"`
	AchievementID string     `json:"achievement_id"`
	Level         int        `json:"level"`
	Progress      int        `json:"progress"`
	UnlockedAt    *time.Time `json:"unlocked_at,omitempty"`
}

// League represents a league tier
type League struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	IconURL          string `json:"icon_url,omitempty"`
	MinXPToPromote   int    `json:"min_xp_to_promote"`
	DemotionZoneSize int    `json:"demotion_zone_size"`
}

// LeagueSeason represents a weekly league season
type LeagueSeason struct {
	ID        uuid.UUID `json:"id"`
	LeagueID  int       `json:"league_id"`
	WeekStart time.Time `json:"week_start"`
	WeekEnd   time.Time `json:"week_end"`
}

// UserLeague represents a user's participation in a league
type UserLeague struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	SeasonID  uuid.UUID  `json:"season_id"`
	XPEarned  int        `json:"xp_earned"`
	Rank      int        `json:"rank"`
	Promoted  bool       `json:"promoted"`
	Demoted   bool       `json:"demoted"`
	User      *User      `json:"user,omitempty"`
}

// FriendQuest represents a friend quest
type FriendQuest struct {
	ID             uuid.UUID `json:"id"`
	User1ID        uuid.UUID `json:"user1_id"`
	User2ID        uuid.UUID `json:"user2_id"`
	QuestType      string    `json:"quest_type"`
	TargetValue    int       `json:"target_value"`
	User1Progress  int       `json:"user1_progress"`
	User2Progress  int       `json:"user2_progress"`
	StartsAt       time.Time `json:"starts_at"`
	EndsAt         time.Time `json:"ends_at"`
	Completed      bool      `json:"completed"`
	RewardsClaimed bool      `json:"rewards_claimed"`
}

// FriendStreak represents a shared streak with a friend
type FriendStreak struct {
	ID             uuid.UUID `json:"id"`
	User1ID        uuid.UUID `json:"user1_id"`
	User2ID        uuid.UUID `json:"user2_id"`
	StreakDays     int       `json:"streak_days"`
	StartedAt      time.Time `json:"started_at"`
	LastBothActive time.Time `json:"last_both_active"`
}

// Notification represents a user notification
type Notification struct {
	ID        uuid.UUID              `json:"id"`
	UserID    uuid.UUID              `json:"user_id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Read      bool                   `json:"read"`
	CreatedAt time.Time              `json:"created_at"`
}

// StoryElementType defines the type of story element
type StoryElementType string

const (
	ElementTypeHeader       StoryElementType = "header"
	ElementTypeLine         StoryElementType = "line"
	ElementTypeNarration    StoryElementType = "narration"    // Narrator text without speaker
	ElementTypeMultiChoice  StoryElementType = "multiple_choice"
	ElementTypeSelectPhrase StoryElementType = "select_phrase"
	ElementTypeSelectWord   StoryElementType = "select_word"   // New: tap word that means X
	ElementTypeWhatNext     StoryElementType = "what_next"     // New: What comes next?
	ElementTypeArrange      StoryElementType = "arrange"
	ElementTypeMatch        StoryElementType = "match"
	ElementTypePointPhrase  StoryElementType = "point_to_phrase"
	ElementTypeTapComplete  StoryElementType = "tap_complete"
)

// Story represents an interactive story
type Story struct {
	ID               uuid.UUID        `json:"id"`
	CourseID         uuid.UUID        `json:"course_id"`
	ExternalID       string           `json:"external_id,omitempty"`
	Title            string           `json:"title"`
	TitleTranslation string           `json:"title_translation,omitempty"`
	IllustrationURL  string           `json:"illustration_url,omitempty"`
	SetID            int              `json:"set_id"`
	SetPosition      int              `json:"set_position"`
	Difficulty       int              `json:"difficulty"`
	CEFRLevel        string           `json:"cefr_level,omitempty"`
	DurationSeconds  int              `json:"duration_seconds"`
	XPReward         int              `json:"xp_reward"`
	Characters       []StoryCharacter `json:"characters,omitempty"`
	Elements         []StoryElement   `json:"elements,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
}

// StoryCharacter represents a character in a story
type StoryCharacter struct {
	ID          uuid.UUID `json:"id"`
	StoryID     uuid.UUID `json:"story_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	VoiceID     string    `json:"voice_id,omitempty"`
	Position    int       `json:"position"`
}

// StoryElement represents a single element in a story
type StoryElement struct {
	ID            uuid.UUID        `json:"id"`
	StoryID       uuid.UUID        `json:"story_id"`
	Position      int              `json:"position"`
	ElementType   StoryElementType `json:"element_type"`
	SpeakerID     *uuid.UUID       `json:"speaker_id,omitempty"`
	Speaker       *StoryCharacter  `json:"speaker,omitempty"`
	Text          string           `json:"text,omitempty"`
	Translation   string           `json:"translation,omitempty"`
	AudioURL      string           `json:"audio_url,omitempty"`
	AudioTiming   []AudioTiming    `json:"audio_timing,omitempty"`
	Tokens        []WordToken      `json:"tokens,omitempty"` // Tokenized text for tappable word hints
	ChallengeData *ChallengeData   `json:"challenge_data,omitempty"`
}

// WordToken represents a word with optional translation hint for tappable words
type WordToken struct {
	Word        string `json:"word"`
	Translation string `json:"translation,omitempty"` // For tappable hint words
	IsTarget    bool   `json:"is_target,omitempty"`   // Is this the correct answer in select_word
	IsTappable  bool   `json:"is_tappable,omitempty"` // Can be tapped for hint
}

// AudioTiming represents timing for a word in audio
type AudioTiming struct {
	Start    int `json:"start"`    // milliseconds from audio start
	Duration int `json:"duration"` // milliseconds
}

// ChallengeData represents challenge-specific data
type ChallengeData struct {
	// Common
	Prompt        string   `json:"prompt,omitempty"`
	Question      string   `json:"question,omitempty"`       // Question text (e.g., "Vikram says his day was")
	CorrectAnswer string   `json:"correct_answer,omitempty"`
	CorrectIndex  int      `json:"correct_index,omitempty"`  // Index of correct option
	Options       []string `json:"options,omitempty"`

	// Feedback
	FeedbackCorrect   string `json:"feedback_correct,omitempty"`
	FeedbackIncorrect string `json:"feedback_incorrect,omitempty"`

	// SelectPhrase: highlighted phrase boundaries
	PhraseStart int `json:"phrase_start,omitempty"`
	PhraseEnd   int `json:"phrase_end,omitempty"`

	// SelectWord: word tokens for tap-to-select challenges
	SentenceWords   []WordToken `json:"sentence_words,omitempty"`   // Tokenized sentence for word selection
	TargetWordIndex int         `json:"target_word_index,omitempty"` // Which word is correct
	TargetMeaning   string      `json:"target_meaning,omitempty"`    // The meaning hint (e.g., "thin" in "choose word meaning thin")

	// Arrange: word segments to reorder
	ArrangeWords []string         `json:"arrange_words,omitempty"` // Simple word array for arrange
	Segments     []ArrangeSegment `json:"segments,omitempty"`

	// Match: pairs to match
	Pairs []MatchPair `json:"pairs,omitempty"`
}

// ArrangeSegment represents a word/phrase segment for arrange challenges
type ArrangeSegment struct {
	Text        string `json:"text"`
	Translation string `json:"translation,omitempty"`
	Position    int    `json:"position"` // correct position
}

// MatchPair represents a word-translation pair for matching
type MatchPair struct {
	Word        string `json:"word"`
	Translation string `json:"translation"`
}

// StorySet represents a collection of stories
type StorySet struct {
	ID                int       `json:"id"`
	CourseID          uuid.UUID `json:"course_id"`
	Name              string    `json:"name"`
	Description       string    `json:"description,omitempty"`
	Position          int       `json:"position"`
	UnlockRequirement string    `json:"unlock_requirement,omitempty"`
	IconURL           string    `json:"icon_url,omitempty"`
	Stories           []Story   `json:"stories,omitempty"`
	StoriesCount      int       `json:"stories_count"`
	CompletedCount    int       `json:"completed_count,omitempty"`
}

// UserStory represents a user's completion of a story
type UserStory struct {
	UserID              uuid.UUID  `json:"user_id"`
	StoryID             uuid.UUID  `json:"story_id"`
	StartedAt           *time.Time `json:"started_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	Completed           bool       `json:"completed"`
	XPEarned            int        `json:"xp_earned"`
	MistakesCount       int        `json:"mistakes_count"`
	ListenModeCompleted bool       `json:"listen_mode_completed"`
}

// UserStoryProgress represents element-level progress
type UserStoryProgress struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	StoryID     uuid.UUID  `json:"story_id"`
	ElementID   uuid.UUID  `json:"element_id"`
	Completed   bool       `json:"completed"`
	Correct     *bool      `json:"correct,omitempty"`
	Attempts    int        `json:"attempts"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
