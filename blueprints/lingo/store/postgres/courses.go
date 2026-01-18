package postgres

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CourseStore implements store.CourseStore
type CourseStore struct {
	pool *pgxpool.Pool
}

// ListLanguages lists all languages
func (s *CourseStore) ListLanguages(ctx context.Context) ([]store.Language, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, native_name, flag_emoji, rtl, enabled
		FROM languages WHERE enabled = true ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("query languages: %w", err)
	}
	defer rows.Close()

	var languages []store.Language
	for rows.Next() {
		var l store.Language
		if err := rows.Scan(&l.ID, &l.Name, &l.NativeName, &l.FlagEmoji, &l.RTL, &l.Enabled); err != nil {
			return nil, fmt.Errorf("scan language: %w", err)
		}
		languages = append(languages, l)
	}
	return languages, nil
}

// ListCourses lists courses available from a language
func (s *CourseStore) ListCourses(ctx context.Context, fromLang string) ([]store.Course, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, from_language_id, learning_language_id, title, description, total_units, cefr_level, enabled
		FROM courses WHERE from_language_id = $1 AND enabled = true ORDER BY title
	`, fromLang)
	if err != nil {
		return nil, fmt.Errorf("query courses: %w", err)
	}
	defer rows.Close()

	var courses []store.Course
	for rows.Next() {
		var c store.Course
		if err := rows.Scan(&c.ID, &c.FromLanguageID, &c.LearningLanguageID, &c.Title, &c.Description, &c.TotalUnits, &c.CEFRLevel, &c.Enabled); err != nil {
			return nil, fmt.Errorf("scan course: %w", err)
		}
		courses = append(courses, c)
	}
	return courses, nil
}

// GetCourse gets a course by ID
func (s *CourseStore) GetCourse(ctx context.Context, id uuid.UUID) (*store.Course, error) {
	course := &store.Course{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, from_language_id, learning_language_id, title, description, total_units, cefr_level, enabled
		FROM courses WHERE id = $1
	`, id).Scan(&course.ID, &course.FromLanguageID, &course.LearningLanguageID, &course.Title, &course.Description, &course.TotalUnits, &course.CEFRLevel, &course.Enabled)
	if err != nil {
		return nil, fmt.Errorf("query course: %w", err)
	}
	return course, nil
}

// GetCoursePath gets the learning path for a course
func (s *CourseStore) GetCoursePath(ctx context.Context, courseID uuid.UUID) ([]store.Unit, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, course_id, position, title, description, guidebook_content, icon_url
		FROM units WHERE course_id = $1 ORDER BY position
	`, courseID)
	if err != nil {
		return nil, fmt.Errorf("query units: %w", err)
	}
	defer rows.Close()

	var units []store.Unit
	for rows.Next() {
		var u store.Unit
		if err := rows.Scan(&u.ID, &u.CourseID, &u.Position, &u.Title, &u.Description, &u.GuidebookContent, &u.IconURL); err != nil {
			return nil, fmt.Errorf("scan unit: %w", err)
		}

		// Get skills for this unit
		skills, err := s.getSkillsForUnit(ctx, u.ID)
		if err != nil {
			return nil, fmt.Errorf("get skills: %w", err)
		}
		u.Skills = skills
		units = append(units, u)
	}
	return units, nil
}

func (s *CourseStore) getSkillsForUnit(ctx context.Context, unitID uuid.UUID) ([]store.Skill, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, unit_id, position, name, icon_name, levels, lexemes_count
		FROM skills WHERE unit_id = $1 ORDER BY position
	`, unitID)
	if err != nil {
		return nil, fmt.Errorf("query skills: %w", err)
	}
	defer rows.Close()

	var skills []store.Skill
	for rows.Next() {
		var sk store.Skill
		if err := rows.Scan(&sk.ID, &sk.UnitID, &sk.Position, &sk.Name, &sk.IconName, &sk.Levels, &sk.LexemesCount); err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		skills = append(skills, sk)
	}
	return skills, nil
}

// GetUnit gets a unit by ID
func (s *CourseStore) GetUnit(ctx context.Context, id uuid.UUID) (*store.Unit, error) {
	unit := &store.Unit{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, course_id, position, title, description, guidebook_content, icon_url
		FROM units WHERE id = $1
	`, id).Scan(&unit.ID, &unit.CourseID, &unit.Position, &unit.Title, &unit.Description, &unit.GuidebookContent, &unit.IconURL)
	if err != nil {
		return nil, fmt.Errorf("query unit: %w", err)
	}
	skills, err := s.getSkillsForUnit(ctx, unit.ID)
	if err != nil {
		return nil, err
	}
	unit.Skills = skills
	return unit, nil
}

// GetSkill gets a skill by ID
func (s *CourseStore) GetSkill(ctx context.Context, id uuid.UUID) (*store.Skill, error) {
	skill := &store.Skill{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, unit_id, position, name, icon_name, levels, lexemes_count
		FROM skills WHERE id = $1
	`, id).Scan(&skill.ID, &skill.UnitID, &skill.Position, &skill.Name, &skill.IconName, &skill.Levels, &skill.LexemesCount)
	if err != nil {
		return nil, fmt.Errorf("query skill: %w", err)
	}
	return skill, nil
}

// GetLesson gets a lesson by ID
func (s *CourseStore) GetLesson(ctx context.Context, id uuid.UUID) (*store.Lesson, error) {
	lesson := &store.Lesson{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, skill_id, level, position, exercise_count
		FROM lessons WHERE id = $1
	`, id).Scan(&lesson.ID, &lesson.SkillID, &lesson.Level, &lesson.Position, &lesson.ExerciseCount)
	if err != nil {
		return nil, fmt.Errorf("query lesson: %w", err)
	}
	return lesson, nil
}

// GetLessonsBySkill returns all lessons for a skill
func (s *CourseStore) GetLessonsBySkill(ctx context.Context, skillID uuid.UUID) ([]store.Lesson, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, skill_id, level, position, exercise_count
		FROM lessons WHERE skill_id = $1 ORDER BY level, position
	`, skillID)
	if err != nil {
		return nil, fmt.Errorf("query lessons: %w", err)
	}
	defer rows.Close()

	var lessons []store.Lesson
	for rows.Next() {
		var lesson store.Lesson
		if err := rows.Scan(&lesson.ID, &lesson.SkillID, &lesson.Level, &lesson.Position, &lesson.ExerciseCount); err != nil {
			return nil, fmt.Errorf("scan lesson: %w", err)
		}
		lessons = append(lessons, lesson)
	}
	return lessons, nil
}

// GetExercises gets exercises for a lesson
func (s *CourseStore) GetExercises(ctx context.Context, lessonID uuid.UUID) ([]store.Exercise, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, lesson_id, type, prompt, correct_answer, choices, audio_url, image_url, hints, difficulty
		FROM exercises WHERE lesson_id = $1 ORDER BY RANDOM()
	`, lessonID)
	if err != nil {
		return nil, fmt.Errorf("query exercises: %w", err)
	}
	defer rows.Close()

	var exercises []store.Exercise
	for rows.Next() {
		var e store.Exercise
		if err := rows.Scan(&e.ID, &e.LessonID, &e.Type, &e.Prompt, &e.CorrectAnswer, &e.Choices, &e.AudioURL, &e.ImageURL, &e.Hints, &e.Difficulty); err != nil {
			return nil, fmt.Errorf("scan exercise: %w", err)
		}
		exercises = append(exercises, e)
	}
	return exercises, nil
}

// GetStories gets stories for a course
func (s *CourseStore) GetStories(ctx context.Context, courseID uuid.UUID) ([]store.Story, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, course_id, external_id, title, title_translation, illustration_url,
		       set_id, set_position, difficulty, cefr_level, duration_seconds, xp_reward, created_at
		FROM stories WHERE course_id = $1 ORDER BY set_id, set_position, difficulty
	`, courseID)
	if err != nil {
		return nil, fmt.Errorf("query stories: %w", err)
	}
	defer rows.Close()

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

// GetLexemesByCourse returns all lexemes for a course
func (s *CourseStore) GetLexemesByCourse(ctx context.Context, courseID uuid.UUID) ([]store.Lexeme, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, course_id, word, translation, pos, COALESCE(audio_url, ''), COALESCE(image_url, ''),
		       COALESCE(example_sentence, ''), COALESCE(example_translation, '')
		FROM lexemes WHERE course_id = $1 ORDER BY word
	`, courseID)
	if err != nil {
		return nil, fmt.Errorf("query lexemes: %w", err)
	}
	defer rows.Close()

	var lexemes []store.Lexeme
	for rows.Next() {
		var lex store.Lexeme
		if err := rows.Scan(&lex.ID, &lex.CourseID, &lex.Word, &lex.Translation, &lex.POS,
			&lex.AudioURL, &lex.ImageURL, &lex.ExampleSentence, &lex.ExampleTranslation); err != nil {
			return nil, fmt.Errorf("scan lexeme: %w", err)
		}
		lexemes = append(lexemes, lex)
	}
	return lexemes, nil
}

// GetStory gets a story by ID
func (s *CourseStore) GetStory(ctx context.Context, id uuid.UUID) (*store.Story, error) {
	story := &store.Story{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, course_id, external_id, title, title_translation, illustration_url,
		       set_id, set_position, difficulty, cefr_level, duration_seconds, xp_reward, created_at
		FROM stories WHERE id = $1
	`, id).Scan(&story.ID, &story.CourseID, &story.ExternalID, &story.Title, &story.TitleTranslation,
		&story.IllustrationURL, &story.SetID, &story.SetPosition, &story.Difficulty,
		&story.CEFRLevel, &story.DurationSeconds, &story.XPReward, &story.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("query story: %w", err)
	}
	return story, nil
}
