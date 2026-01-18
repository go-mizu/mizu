package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// StoryStore implements store.StoryStore for SQLite
type StoryStore struct {
	db *sql.DB
}

// GetStorySets returns all story sets for a course
func (s *StoryStore) GetStorySets(ctx context.Context, courseID uuid.UUID) ([]store.StorySet, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, course_id, name, description, position, unlock_requirement, icon_url
		FROM story_sets
		WHERE course_id = ?
		ORDER BY position
	`, courseID.String())
	if err != nil {
		return nil, fmt.Errorf("query story sets: %w", err)
	}
	defer rows.Close()

	var sets []store.StorySet
	for rows.Next() {
		var set store.StorySet
		var courseIDStr string
		var description, unlockReq, iconURL sql.NullString
		if err := rows.Scan(&set.ID, &courseIDStr, &set.Name, &description, &set.Position, &unlockReq, &iconURL); err != nil {
			return nil, fmt.Errorf("scan story set: %w", err)
		}
		set.CourseID, _ = uuid.Parse(courseIDStr)
		set.Description = description.String
		set.UnlockRequirement = unlockReq.String
		set.IconURL = iconURL.String

		// Get story count for this set
		var count int
		err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM stories WHERE course_id = ? AND set_id = ?`,
			courseID.String(), set.ID).Scan(&count)
		if err == nil {
			set.StoriesCount = count
		}

		sets = append(sets, set)
	}

	return sets, nil
}

// GetStories returns all stories for a course
func (s *StoryStore) GetStories(ctx context.Context, courseID uuid.UUID) ([]store.Story, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, course_id, external_id, title, title_translation, illustration_url,
		       set_id, set_position, difficulty, cefr_level, duration_seconds, xp_reward, created_at
		FROM stories
		WHERE course_id = ?
		ORDER BY set_id, set_position, difficulty
	`, courseID.String())
	if err != nil {
		return nil, fmt.Errorf("query stories: %w", err)
	}
	defer rows.Close()

	return s.scanStories(rows)
}

// GetStoriesBySet returns stories for a specific set
func (s *StoryStore) GetStoriesBySet(ctx context.Context, courseID uuid.UUID, setID int) ([]store.Story, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, course_id, external_id, title, title_translation, illustration_url,
		       set_id, set_position, difficulty, cefr_level, duration_seconds, xp_reward, created_at
		FROM stories
		WHERE course_id = ? AND set_id = ?
		ORDER BY set_position, difficulty
	`, courseID.String(), setID)
	if err != nil {
		return nil, fmt.Errorf("query stories by set: %w", err)
	}
	defer rows.Close()

	return s.scanStories(rows)
}

func (s *StoryStore) scanStories(rows *sql.Rows) ([]store.Story, error) {
	var stories []store.Story
	for rows.Next() {
		var st store.Story
		var idStr, courseIDStr string
		var externalID, titleTranslation, illustrationURL, cefrLevel sql.NullString
		var createdAt sql.NullTime

		if err := rows.Scan(&idStr, &courseIDStr, &externalID, &st.Title, &titleTranslation,
			&illustrationURL, &st.SetID, &st.SetPosition, &st.Difficulty,
			&cefrLevel, &st.DurationSeconds, &st.XPReward, &createdAt); err != nil {
			return nil, fmt.Errorf("scan story: %w", err)
		}

		st.ID, _ = uuid.Parse(idStr)
		st.CourseID, _ = uuid.Parse(courseIDStr)
		st.ExternalID = externalID.String
		st.TitleTranslation = titleTranslation.String
		st.IllustrationURL = illustrationURL.String
		st.CEFRLevel = cefrLevel.String
		if createdAt.Valid {
			st.CreatedAt = createdAt.Time
		}

		stories = append(stories, st)
	}

	return stories, nil
}

// GetStory returns a single story with its characters and elements
func (s *StoryStore) GetStory(ctx context.Context, id uuid.UUID) (*store.Story, error) {
	var st store.Story
	var idStr, courseIDStr string
	var externalID, titleTranslation, illustrationURL, cefrLevel sql.NullString
	var createdAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, course_id, external_id, title, title_translation, illustration_url,
		       set_id, set_position, difficulty, cefr_level, duration_seconds, xp_reward, created_at
		FROM stories
		WHERE id = ?
	`, id.String()).Scan(&idStr, &courseIDStr, &externalID, &st.Title, &titleTranslation,
		&illustrationURL, &st.SetID, &st.SetPosition, &st.Difficulty,
		&cefrLevel, &st.DurationSeconds, &st.XPReward, &createdAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("story not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("query story: %w", err)
	}

	st.ID, _ = uuid.Parse(idStr)
	st.CourseID, _ = uuid.Parse(courseIDStr)
	st.ExternalID = externalID.String
	st.TitleTranslation = titleTranslation.String
	st.IllustrationURL = illustrationURL.String
	st.CEFRLevel = cefrLevel.String
	if createdAt.Valid {
		st.CreatedAt = createdAt.Time
	}

	// Load characters
	chars, err := s.GetStoryCharacters(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get story characters: %w", err)
	}
	st.Characters = chars

	// Load elements
	elems, err := s.GetStoryElements(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get story elements: %w", err)
	}
	st.Elements = elems

	return &st, nil
}

// GetStoryElements returns all elements for a story
func (s *StoryStore) GetStoryElements(ctx context.Context, storyID uuid.UUID) ([]store.StoryElement, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, story_id, position, element_type, speaker_id, text, translation, audio_url, audio_timing, challenge_data
		FROM story_elements
		WHERE story_id = ?
		ORDER BY position
	`, storyID.String())
	if err != nil {
		return nil, fmt.Errorf("query story elements: %w", err)
	}
	defer rows.Close()

	var elements []store.StoryElement
	for rows.Next() {
		var elem store.StoryElement
		var idStr, storyIDStr string
		var speakerIDStr, text, translation, audioURL, audioTimingStr, challengeDataStr sql.NullString

		if err := rows.Scan(&idStr, &storyIDStr, &elem.Position, &elem.ElementType,
			&speakerIDStr, &text, &translation, &audioURL, &audioTimingStr, &challengeDataStr); err != nil {
			return nil, fmt.Errorf("scan story element: %w", err)
		}

		elem.ID, _ = uuid.Parse(idStr)
		elem.StoryID, _ = uuid.Parse(storyIDStr)
		if speakerIDStr.Valid {
			speakerID, _ := uuid.Parse(speakerIDStr.String)
			elem.SpeakerID = &speakerID
		}
		elem.Text = text.String
		elem.Translation = translation.String
		elem.AudioURL = audioURL.String

		// Parse audio timing JSON
		if audioTimingStr.Valid && audioTimingStr.String != "" {
			var timing []store.AudioTiming
			if err := json.Unmarshal([]byte(audioTimingStr.String), &timing); err == nil {
				elem.AudioTiming = timing
			}
		}

		// Parse challenge data JSON
		if challengeDataStr.Valid && challengeDataStr.String != "" {
			var cd store.ChallengeData
			if err := json.Unmarshal([]byte(challengeDataStr.String), &cd); err == nil {
				elem.ChallengeData = &cd
			}
		}

		elements = append(elements, elem)
	}

	return elements, nil
}

// GetStoryCharacters returns all characters for a story
func (s *StoryStore) GetStoryCharacters(ctx context.Context, storyID uuid.UUID) ([]store.StoryCharacter, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, story_id, name, display_name, avatar_url, voice_id, position
		FROM story_characters
		WHERE story_id = ?
		ORDER BY position
	`, storyID.String())
	if err != nil {
		return nil, fmt.Errorf("query story characters: %w", err)
	}
	defer rows.Close()

	var characters []store.StoryCharacter
	for rows.Next() {
		var char store.StoryCharacter
		var idStr, storyIDStr string
		var displayName, avatarURL, voiceID sql.NullString

		if err := rows.Scan(&idStr, &storyIDStr, &char.Name, &displayName, &avatarURL, &voiceID, &char.Position); err != nil {
			return nil, fmt.Errorf("scan story character: %w", err)
		}

		char.ID, _ = uuid.Parse(idStr)
		char.StoryID, _ = uuid.Parse(storyIDStr)
		char.DisplayName = displayName.String
		char.AvatarURL = avatarURL.String
		char.VoiceID = voiceID.String

		characters = append(characters, char)
	}

	return characters, nil
}

// GetUserStory returns user progress on a story
func (s *StoryStore) GetUserStory(ctx context.Context, userID, storyID uuid.UUID) (*store.UserStory, error) {
	var us store.UserStory
	var userIDStr, storyIDStr string
	var startedAt, completedAt sql.NullTime
	var completed, listenModeCompleted int

	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, story_id, started_at, completed_at, completed, xp_earned, mistakes_count, listen_mode_completed
		FROM user_stories
		WHERE user_id = ? AND story_id = ?
	`, userID.String(), storyID.String()).Scan(&userIDStr, &storyIDStr, &startedAt, &completedAt,
		&completed, &us.XPEarned, &us.MistakesCount, &listenModeCompleted)

	if err == sql.ErrNoRows {
		return nil, nil // Not started yet
	}
	if err != nil {
		return nil, fmt.Errorf("query user story: %w", err)
	}

	us.UserID, _ = uuid.Parse(userIDStr)
	us.StoryID, _ = uuid.Parse(storyIDStr)
	if startedAt.Valid {
		us.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		us.CompletedAt = &completedAt.Time
	}
	us.Completed = completed == 1
	us.ListenModeCompleted = listenModeCompleted == 1

	return &us, nil
}

// GetUserStories returns all user stories for a course
func (s *StoryStore) GetUserStories(ctx context.Context, userID, courseID uuid.UUID) ([]store.UserStory, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT us.user_id, us.story_id, us.started_at, us.completed_at, us.completed, us.xp_earned, us.mistakes_count, us.listen_mode_completed
		FROM user_stories us
		JOIN stories st ON us.story_id = st.id
		WHERE us.user_id = ? AND st.course_id = ?
	`, userID.String(), courseID.String())
	if err != nil {
		return nil, fmt.Errorf("query user stories: %w", err)
	}
	defer rows.Close()

	var userStories []store.UserStory
	for rows.Next() {
		var us store.UserStory
		var userIDStr, storyIDStr string
		var startedAt, completedAt sql.NullTime
		var completed, listenModeCompleted int

		if err := rows.Scan(&userIDStr, &storyIDStr, &startedAt, &completedAt,
			&completed, &us.XPEarned, &us.MistakesCount, &listenModeCompleted); err != nil {
			return nil, fmt.Errorf("scan user story: %w", err)
		}

		us.UserID, _ = uuid.Parse(userIDStr)
		us.StoryID, _ = uuid.Parse(storyIDStr)
		if startedAt.Valid {
			us.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			us.CompletedAt = &completedAt.Time
		}
		us.Completed = completed == 1
		us.ListenModeCompleted = listenModeCompleted == 1

		userStories = append(userStories, us)
	}

	return userStories, nil
}

// StartStory records that a user started a story
func (s *StoryStore) StartStory(ctx context.Context, userID, storyID uuid.UUID) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_stories (user_id, story_id, started_at, completed, xp_earned, mistakes_count, listen_mode_completed)
		VALUES (?, ?, ?, 0, 0, 0, 0)
		ON CONFLICT(user_id, story_id) DO NOTHING
	`, userID.String(), storyID.String(), now)
	if err != nil {
		return fmt.Errorf("start story: %w", err)
	}
	return nil
}

// CompleteStory marks a story as completed
func (s *StoryStore) CompleteStory(ctx context.Context, userID, storyID uuid.UUID, xp, mistakes int) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE user_stories
		SET completed = 1, completed_at = ?, xp_earned = ?, mistakes_count = ?
		WHERE user_id = ? AND story_id = ?
	`, now, xp, mistakes, userID.String(), storyID.String())
	if err != nil {
		return fmt.Errorf("complete story: %w", err)
	}
	return nil
}

// RecordElementProgress records progress on a story element
func (s *StoryStore) RecordElementProgress(ctx context.Context, progress *store.UserStoryProgress) error {
	correctVal := sql.NullInt64{}
	if progress.Correct != nil {
		correctVal.Valid = true
		if *progress.Correct {
			correctVal.Int64 = 1
		} else {
			correctVal.Int64 = 0
		}
	}

	var completedAt sql.NullTime
	if progress.CompletedAt != nil {
		completedAt.Valid = true
		completedAt.Time = *progress.CompletedAt
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_story_progress (id, user_id, story_id, element_id, completed, correct, attempts, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, story_id, element_id) DO UPDATE SET
			completed = excluded.completed,
			correct = excluded.correct,
			attempts = excluded.attempts,
			completed_at = excluded.completed_at
	`, progress.ID.String(), progress.UserID.String(), progress.StoryID.String(), progress.ElementID.String(),
		boolToInt(progress.Completed), correctVal, progress.Attempts, completedAt)
	if err != nil {
		return fmt.Errorf("record element progress: %w", err)
	}
	return nil
}

// GetStoryProgress returns all element progress for a user on a story
func (s *StoryStore) GetStoryProgress(ctx context.Context, userID, storyID uuid.UUID) ([]store.UserStoryProgress, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, story_id, element_id, completed, correct, attempts, completed_at
		FROM user_story_progress
		WHERE user_id = ? AND story_id = ?
	`, userID.String(), storyID.String())
	if err != nil {
		return nil, fmt.Errorf("query story progress: %w", err)
	}
	defer rows.Close()

	var progress []store.UserStoryProgress
	for rows.Next() {
		var p store.UserStoryProgress
		var idStr, userIDStr, storyIDStr, elementIDStr string
		var completed int
		var correct sql.NullInt64
		var completedAt sql.NullTime

		if err := rows.Scan(&idStr, &userIDStr, &storyIDStr, &elementIDStr,
			&completed, &correct, &p.Attempts, &completedAt); err != nil {
			return nil, fmt.Errorf("scan story progress: %w", err)
		}

		p.ID, _ = uuid.Parse(idStr)
		p.UserID, _ = uuid.Parse(userIDStr)
		p.StoryID, _ = uuid.Parse(storyIDStr)
		p.ElementID, _ = uuid.Parse(elementIDStr)
		p.Completed = completed == 1
		if correct.Valid {
			c := correct.Int64 == 1
			p.Correct = &c
		}
		if completedAt.Valid {
			p.CompletedAt = &completedAt.Time
		}

		progress = append(progress, p)
	}

	return progress, nil
}

// CreateStory creates a new story
func (s *StoryStore) CreateStory(ctx context.Context, story *store.Story) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO stories (id, course_id, external_id, title, title_translation, illustration_url,
		                     set_id, set_position, difficulty, cefr_level, duration_seconds, xp_reward, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, story.ID.String(), story.CourseID.String(), nullString(story.ExternalID), story.Title,
		nullString(story.TitleTranslation), nullString(story.IllustrationURL),
		story.SetID, story.SetPosition, story.Difficulty, nullString(story.CEFRLevel),
		story.DurationSeconds, story.XPReward, time.Now())
	if err != nil {
		return fmt.Errorf("create story: %w", err)
	}
	return nil
}

// CreateStoryCharacter creates a new story character
func (s *StoryStore) CreateStoryCharacter(ctx context.Context, char *store.StoryCharacter) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO story_characters (id, story_id, name, display_name, avatar_url, voice_id, position)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(story_id, name) DO UPDATE SET
			display_name = excluded.display_name,
			avatar_url = excluded.avatar_url,
			voice_id = excluded.voice_id,
			position = excluded.position
	`, char.ID.String(), char.StoryID.String(), char.Name, nullString(char.DisplayName),
		nullString(char.AvatarURL), nullString(char.VoiceID), char.Position)
	if err != nil {
		return fmt.Errorf("create story character: %w", err)
	}
	return nil
}

// CreateStoryElement creates a new story element
func (s *StoryStore) CreateStoryElement(ctx context.Context, elem *store.StoryElement) error {
	var audioTimingJSON, challengeDataJSON []byte
	var err error

	if len(elem.AudioTiming) > 0 {
		audioTimingJSON, err = json.Marshal(elem.AudioTiming)
		if err != nil {
			return fmt.Errorf("marshal audio timing: %w", err)
		}
	}

	if elem.ChallengeData != nil {
		challengeDataJSON, err = json.Marshal(elem.ChallengeData)
		if err != nil {
			return fmt.Errorf("marshal challenge data: %w", err)
		}
	}

	var speakerID sql.NullString
	if elem.SpeakerID != nil {
		speakerID.Valid = true
		speakerID.String = elem.SpeakerID.String()
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO story_elements (id, story_id, position, element_type, speaker_id, text, translation, audio_url, audio_timing, challenge_data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, elem.ID.String(), elem.StoryID.String(), elem.Position, string(elem.ElementType),
		speakerID, nullString(elem.Text), nullString(elem.Translation), nullString(elem.AudioURL),
		nullString(string(audioTimingJSON)), nullString(string(challengeDataJSON)))
	if err != nil {
		return fmt.Errorf("create story element: %w", err)
	}
	return nil
}

// CreateStorySet creates a new story set
func (s *StoryStore) CreateStorySet(ctx context.Context, set *store.StorySet) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO story_sets (id, course_id, name, description, position, unlock_requirement, icon_url)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			position = excluded.position,
			unlock_requirement = excluded.unlock_requirement,
			icon_url = excluded.icon_url
	`, set.ID, set.CourseID.String(), set.Name, nullString(set.Description),
		set.Position, nullString(set.UnlockRequirement), nullString(set.IconURL))
	if err != nil {
		return fmt.Errorf("create story set: %w", err)
	}
	return nil
}

// DeleteStoriesByCourse deletes all stories for a course
func (s *StoryStore) DeleteStoriesByCourse(ctx context.Context, courseID uuid.UUID) error {
	// Delete in order due to foreign keys
	// First get story IDs
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM stories WHERE course_id = ?`, courseID.String())
	if err != nil {
		return fmt.Errorf("query stories: %w", err)
	}

	var storyIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		storyIDs = append(storyIDs, id)
	}
	rows.Close()

	// Delete related records for each story
	for _, storyID := range storyIDs {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM user_story_progress WHERE story_id = ?`, storyID); err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `DELETE FROM user_stories WHERE story_id = ?`, storyID); err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `DELETE FROM story_elements WHERE story_id = ?`, storyID); err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `DELETE FROM story_characters WHERE story_id = ?`, storyID); err != nil {
			return err
		}
	}

	// Delete stories
	if _, err := s.db.ExecContext(ctx, `DELETE FROM stories WHERE course_id = ?`, courseID.String()); err != nil {
		return fmt.Errorf("delete stories: %w", err)
	}

	// Delete story sets
	if _, err := s.db.ExecContext(ctx, `DELETE FROM story_sets WHERE course_id = ?`, courseID.String()); err != nil {
		return fmt.Errorf("delete story sets: %w", err)
	}

	return nil
}

// Helper functions
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

