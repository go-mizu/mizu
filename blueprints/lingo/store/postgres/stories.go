package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StoryStore implements store.StoryStore for PostgreSQL
type StoryStore struct {
	pool *pgxpool.Pool
}

// GetStorySets returns all story sets for a course
func (s *StoryStore) GetStorySets(ctx context.Context, courseID uuid.UUID) ([]store.StorySet, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, course_id, name, description, position, unlock_requirement, icon_url
		FROM story_sets
		WHERE course_id = $1
		ORDER BY position
	`, courseID)
	if err != nil {
		return nil, fmt.Errorf("query story sets: %w", err)
	}
	defer rows.Close()

	var sets []store.StorySet
	for rows.Next() {
		var set store.StorySet
		if err := rows.Scan(&set.ID, &set.CourseID, &set.Name, &set.Description, &set.Position, &set.UnlockRequirement, &set.IconURL); err != nil {
			return nil, fmt.Errorf("scan story set: %w", err)
		}

		// Get story count for this set
		var count int
		_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM stories WHERE course_id = $1 AND set_id = $2`,
			courseID, set.ID).Scan(&count)
		set.StoriesCount = count

		sets = append(sets, set)
	}

	return sets, nil
}

// GetStories returns all stories for a course
func (s *StoryStore) GetStories(ctx context.Context, courseID uuid.UUID) ([]store.Story, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, course_id, external_id, title, title_translation, illustration_url,
		       set_id, set_position, difficulty, cefr_level, duration_seconds, xp_reward, created_at
		FROM stories
		WHERE course_id = $1
		ORDER BY set_id, set_position, difficulty
	`, courseID)
	if err != nil {
		return nil, fmt.Errorf("query stories: %w", err)
	}
	defer rows.Close()

	return s.scanStories(rows)
}

// GetStoriesBySet returns stories for a specific set
func (s *StoryStore) GetStoriesBySet(ctx context.Context, courseID uuid.UUID, setID int) ([]store.Story, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, course_id, external_id, title, title_translation, illustration_url,
		       set_id, set_position, difficulty, cefr_level, duration_seconds, xp_reward, created_at
		FROM stories
		WHERE course_id = $1 AND set_id = $2
		ORDER BY set_position, difficulty
	`, courseID, setID)
	if err != nil {
		return nil, fmt.Errorf("query stories by set: %w", err)
	}
	defer rows.Close()

	return s.scanStories(rows)
}

type storyRows interface {
	Next() bool
	Scan(dest ...interface{}) error
}

func (s *StoryStore) scanStories(rows storyRows) ([]store.Story, error) {
	var stories []store.Story
	for rows.Next() {
		var st store.Story
		if err := rows.Scan(&st.ID, &st.CourseID, &st.ExternalID, &st.Title, &st.TitleTranslation,
			&st.IllustrationURL, &st.SetID, &st.SetPosition, &st.Difficulty,
			&st.CEFRLevel, &st.DurationSeconds, &st.XPReward, &st.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan story: %w", err)
		}
		stories = append(stories, st)
	}
	return stories, nil
}

// GetStory returns a single story with its characters and elements
func (s *StoryStore) GetStory(ctx context.Context, id uuid.UUID) (*store.Story, error) {
	var st store.Story
	err := s.pool.QueryRow(ctx, `
		SELECT id, course_id, external_id, title, title_translation, illustration_url,
		       set_id, set_position, difficulty, cefr_level, duration_seconds, xp_reward, created_at
		FROM stories
		WHERE id = $1
	`, id).Scan(&st.ID, &st.CourseID, &st.ExternalID, &st.Title, &st.TitleTranslation,
		&st.IllustrationURL, &st.SetID, &st.SetPosition, &st.Difficulty,
		&st.CEFRLevel, &st.DurationSeconds, &st.XPReward, &st.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("query story: %w", err)
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
	rows, err := s.pool.Query(ctx, `
		SELECT id, story_id, position, element_type, speaker_id, text, translation, audio_url, audio_timing, challenge_data
		FROM story_elements
		WHERE story_id = $1
		ORDER BY position
	`, storyID)
	if err != nil {
		return nil, fmt.Errorf("query story elements: %w", err)
	}
	defer rows.Close()

	var elements []store.StoryElement
	for rows.Next() {
		var elem store.StoryElement
		var audioTimingJSON, challengeDataJSON []byte

		if err := rows.Scan(&elem.ID, &elem.StoryID, &elem.Position, &elem.ElementType,
			&elem.SpeakerID, &elem.Text, &elem.Translation, &elem.AudioURL, &audioTimingJSON, &challengeDataJSON); err != nil {
			return nil, fmt.Errorf("scan story element: %w", err)
		}

		// Parse audio timing JSON
		if len(audioTimingJSON) > 0 {
			var timing []store.AudioTiming
			if err := json.Unmarshal(audioTimingJSON, &timing); err == nil {
				elem.AudioTiming = timing
			}
		}

		// Parse challenge data JSON
		if len(challengeDataJSON) > 0 {
			var cd store.ChallengeData
			if err := json.Unmarshal(challengeDataJSON, &cd); err == nil {
				elem.ChallengeData = &cd
			}
		}

		elements = append(elements, elem)
	}

	return elements, nil
}

// GetStoryCharacters returns all characters for a story
func (s *StoryStore) GetStoryCharacters(ctx context.Context, storyID uuid.UUID) ([]store.StoryCharacter, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, story_id, name, display_name, avatar_url, voice_id, position
		FROM story_characters
		WHERE story_id = $1
		ORDER BY position
	`, storyID)
	if err != nil {
		return nil, fmt.Errorf("query story characters: %w", err)
	}
	defer rows.Close()

	var characters []store.StoryCharacter
	for rows.Next() {
		var char store.StoryCharacter
		if err := rows.Scan(&char.ID, &char.StoryID, &char.Name, &char.DisplayName, &char.AvatarURL, &char.VoiceID, &char.Position); err != nil {
			return nil, fmt.Errorf("scan story character: %w", err)
		}
		characters = append(characters, char)
	}

	return characters, nil
}

// GetUserStory returns user progress on a story
func (s *StoryStore) GetUserStory(ctx context.Context, userID, storyID uuid.UUID) (*store.UserStory, error) {
	var us store.UserStory
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, story_id, started_at, completed_at, completed, xp_earned, mistakes_count, listen_mode_completed
		FROM user_stories
		WHERE user_id = $1 AND story_id = $2
	`, userID, storyID).Scan(&us.UserID, &us.StoryID, &us.StartedAt, &us.CompletedAt,
		&us.Completed, &us.XPEarned, &us.MistakesCount, &us.ListenModeCompleted)

	if err != nil {
		return nil, nil // Not started yet
	}

	return &us, nil
}

// GetUserStories returns all user stories for a course
func (s *StoryStore) GetUserStories(ctx context.Context, userID, courseID uuid.UUID) ([]store.UserStory, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT us.user_id, us.story_id, us.started_at, us.completed_at, us.completed, us.xp_earned, us.mistakes_count, us.listen_mode_completed
		FROM user_stories us
		JOIN stories st ON us.story_id = st.id
		WHERE us.user_id = $1 AND st.course_id = $2
	`, userID, courseID)
	if err != nil {
		return nil, fmt.Errorf("query user stories: %w", err)
	}
	defer rows.Close()

	var userStories []store.UserStory
	for rows.Next() {
		var us store.UserStory
		if err := rows.Scan(&us.UserID, &us.StoryID, &us.StartedAt, &us.CompletedAt,
			&us.Completed, &us.XPEarned, &us.MistakesCount, &us.ListenModeCompleted); err != nil {
			return nil, fmt.Errorf("scan user story: %w", err)
		}
		userStories = append(userStories, us)
	}

	return userStories, nil
}

// StartStory records that a user started a story
func (s *StoryStore) StartStory(ctx context.Context, userID, storyID uuid.UUID) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_stories (user_id, story_id, started_at, completed, xp_earned, mistakes_count, listen_mode_completed)
		VALUES ($1, $2, $3, false, 0, 0, false)
		ON CONFLICT(user_id, story_id) DO NOTHING
	`, userID, storyID, now)
	if err != nil {
		return fmt.Errorf("start story: %w", err)
	}
	return nil
}

// CompleteStory marks a story as completed
func (s *StoryStore) CompleteStory(ctx context.Context, userID, storyID uuid.UUID, xp, mistakes int) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx, `
		UPDATE user_stories
		SET completed = true, completed_at = $1, xp_earned = $2, mistakes_count = $3
		WHERE user_id = $4 AND story_id = $5
	`, now, xp, mistakes, userID, storyID)
	if err != nil {
		return fmt.Errorf("complete story: %w", err)
	}
	return nil
}

// RecordElementProgress records progress on a story element
func (s *StoryStore) RecordElementProgress(ctx context.Context, progress *store.UserStoryProgress) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_story_progress (id, user_id, story_id, element_id, completed, correct, attempts, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT(user_id, story_id, element_id) DO UPDATE SET
			completed = EXCLUDED.completed,
			correct = EXCLUDED.correct,
			attempts = EXCLUDED.attempts,
			completed_at = EXCLUDED.completed_at
	`, progress.ID, progress.UserID, progress.StoryID, progress.ElementID,
		progress.Completed, progress.Correct, progress.Attempts, progress.CompletedAt)
	if err != nil {
		return fmt.Errorf("record element progress: %w", err)
	}
	return nil
}

// GetStoryProgress returns all element progress for a user on a story
func (s *StoryStore) GetStoryProgress(ctx context.Context, userID, storyID uuid.UUID) ([]store.UserStoryProgress, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, story_id, element_id, completed, correct, attempts, completed_at
		FROM user_story_progress
		WHERE user_id = $1 AND story_id = $2
	`, userID, storyID)
	if err != nil {
		return nil, fmt.Errorf("query story progress: %w", err)
	}
	defer rows.Close()

	var progress []store.UserStoryProgress
	for rows.Next() {
		var p store.UserStoryProgress
		if err := rows.Scan(&p.ID, &p.UserID, &p.StoryID, &p.ElementID,
			&p.Completed, &p.Correct, &p.Attempts, &p.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan story progress: %w", err)
		}
		progress = append(progress, p)
	}

	return progress, nil
}

// CreateStory creates a new story
func (s *StoryStore) CreateStory(ctx context.Context, story *store.Story) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO stories (id, course_id, external_id, title, title_translation, illustration_url,
		                     set_id, set_position, difficulty, cefr_level, duration_seconds, xp_reward, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, story.ID, story.CourseID, story.ExternalID, story.Title,
		story.TitleTranslation, story.IllustrationURL,
		story.SetID, story.SetPosition, story.Difficulty, story.CEFRLevel,
		story.DurationSeconds, story.XPReward, time.Now())
	if err != nil {
		return fmt.Errorf("create story: %w", err)
	}
	return nil
}

// CreateStoryCharacter creates a new story character
func (s *StoryStore) CreateStoryCharacter(ctx context.Context, char *store.StoryCharacter) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO story_characters (id, story_id, name, display_name, avatar_url, voice_id, position)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(story_id, name) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			avatar_url = EXCLUDED.avatar_url,
			voice_id = EXCLUDED.voice_id,
			position = EXCLUDED.position
	`, char.ID, char.StoryID, char.Name, char.DisplayName,
		char.AvatarURL, char.VoiceID, char.Position)
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

	_, err = s.pool.Exec(ctx, `
		INSERT INTO story_elements (id, story_id, position, element_type, speaker_id, text, translation, audio_url, audio_timing, challenge_data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, elem.ID, elem.StoryID, elem.Position, string(elem.ElementType),
		elem.SpeakerID, elem.Text, elem.Translation, elem.AudioURL,
		audioTimingJSON, challengeDataJSON)
	if err != nil {
		return fmt.Errorf("create story element: %w", err)
	}
	return nil
}

// CreateStorySet creates a new story set
func (s *StoryStore) CreateStorySet(ctx context.Context, set *store.StorySet) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO story_sets (id, course_id, name, description, position, unlock_requirement, icon_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			position = EXCLUDED.position,
			unlock_requirement = EXCLUDED.unlock_requirement,
			icon_url = EXCLUDED.icon_url
	`, set.ID, set.CourseID, set.Name, set.Description,
		set.Position, set.UnlockRequirement, set.IconURL)
	if err != nil {
		return fmt.Errorf("create story set: %w", err)
	}
	return nil
}

// DeleteStoriesByCourse deletes all stories for a course
func (s *StoryStore) DeleteStoriesByCourse(ctx context.Context, courseID uuid.UUID) error {
	// Delete in order due to foreign keys
	// First get story IDs
	rows, err := s.pool.Query(ctx, `SELECT id FROM stories WHERE course_id = $1`, courseID)
	if err != nil {
		return fmt.Errorf("query stories: %w", err)
	}

	var storyIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		storyIDs = append(storyIDs, id)
	}
	rows.Close()

	// Delete related records for each story
	for _, storyID := range storyIDs {
		if _, err := s.pool.Exec(ctx, `DELETE FROM user_story_progress WHERE story_id = $1`, storyID); err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, `DELETE FROM user_stories WHERE story_id = $1`, storyID); err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, `DELETE FROM story_elements WHERE story_id = $1`, storyID); err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, `DELETE FROM story_characters WHERE story_id = $1`, storyID); err != nil {
			return err
		}
	}

	// Delete stories
	if _, err := s.pool.Exec(ctx, `DELETE FROM stories WHERE course_id = $1`, courseID); err != nil {
		return fmt.Errorf("delete stories: %w", err)
	}

	// Delete story sets
	if _, err := s.pool.Exec(ctx, `DELETE FROM story_sets WHERE course_id = $1`, courseID); err != nil {
		return fmt.Errorf("delete story sets: %w", err)
	}

	return nil
}
