package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// CourseStore handles course operations
type CourseStore struct {
	db *sql.DB
}

// ListLanguages returns all enabled languages
func (s *CourseStore) ListLanguages(ctx context.Context) ([]store.Language, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, native_name, flag_emoji, rtl, enabled
		FROM languages WHERE enabled = 1 ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var languages []store.Language
	for rows.Next() {
		var lang store.Language
		var rtl, enabled int
		if err := rows.Scan(&lang.ID, &lang.Name, &lang.NativeName, &lang.FlagEmoji, &rtl, &enabled); err != nil {
			return nil, err
		}
		lang.RTL = rtl == 1
		lang.Enabled = enabled == 1
		languages = append(languages, lang)
	}

	return languages, rows.Err()
}

// ListCourses returns courses for a source language
func (s *CourseStore) ListCourses(ctx context.Context, fromLang string) ([]store.Course, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, from_language_id, learning_language_id, title, description, total_units, cefr_level, enabled
		FROM courses WHERE from_language_id = ? AND enabled = 1
	`, fromLang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []store.Course
	for rows.Next() {
		var course store.Course
		var id string
		var enabled int
		if err := rows.Scan(&id, &course.FromLanguageID, &course.LearningLanguageID, &course.Title,
			&course.Description, &course.TotalUnits, &course.CEFRLevel, &enabled); err != nil {
			return nil, err
		}
		course.ID, _ = uuid.Parse(id)
		course.Enabled = enabled == 1
		courses = append(courses, course)
	}

	return courses, rows.Err()
}

// GetCourse returns a course by ID
func (s *CourseStore) GetCourse(ctx context.Context, id uuid.UUID) (*store.Course, error) {
	var course store.Course
	var courseID string
	var enabled int

	err := s.db.QueryRowContext(ctx, `
		SELECT id, from_language_id, learning_language_id, title, description, total_units, cefr_level, enabled
		FROM courses WHERE id = ?
	`, id.String()).Scan(&courseID, &course.FromLanguageID, &course.LearningLanguageID, &course.Title,
		&course.Description, &course.TotalUnits, &course.CEFRLevel, &enabled)
	if err != nil {
		return nil, err
	}

	course.ID, _ = uuid.Parse(courseID)
	course.Enabled = enabled == 1

	return &course, nil
}

// GetCoursePath returns units with skills for a course
func (s *CourseStore) GetCoursePath(ctx context.Context, courseID uuid.UUID) ([]store.Unit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, course_id, position, title, description, guidebook_content, icon_url
		FROM units WHERE course_id = ? ORDER BY position
	`, courseID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var units []store.Unit
	for rows.Next() {
		var unit store.Unit
		var id, cID string
		var guidebookContent, iconURL sql.NullString
		if err := rows.Scan(&id, &cID, &unit.Position, &unit.Title, &unit.Description,
			&guidebookContent, &iconURL); err != nil {
			return nil, err
		}
		unit.ID, _ = uuid.Parse(id)
		unit.CourseID, _ = uuid.Parse(cID)
		unit.GuidebookContent = guidebookContent.String
		unit.IconURL = iconURL.String

		// Get skills for this unit
		skills, err := s.getSkillsForUnit(ctx, unit.ID)
		if err != nil {
			return nil, err
		}
		unit.Skills = skills

		units = append(units, unit)
	}

	return units, rows.Err()
}

func (s *CourseStore) getSkillsForUnit(ctx context.Context, unitID uuid.UUID) ([]store.Skill, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, unit_id, position, name, icon_name, levels, lexemes_count
		FROM skills WHERE unit_id = ? ORDER BY position
	`, unitID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []store.Skill
	for rows.Next() {
		var skill store.Skill
		var id, uID string
		if err := rows.Scan(&id, &uID, &skill.Position, &skill.Name, &skill.IconName,
			&skill.Levels, &skill.LexemesCount); err != nil {
			return nil, err
		}
		skill.ID, _ = uuid.Parse(id)
		skill.UnitID, _ = uuid.Parse(uID)
		skills = append(skills, skill)
	}

	return skills, rows.Err()
}

// GetUnit returns a unit by ID with its skills
func (s *CourseStore) GetUnit(ctx context.Context, id uuid.UUID) (*store.Unit, error) {
	var unit store.Unit
	var unitID, courseID string
	var guidebookContent, iconURL sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, course_id, position, title, description, guidebook_content, icon_url
		FROM units WHERE id = ?
	`, id.String()).Scan(&unitID, &courseID, &unit.Position, &unit.Title, &unit.Description,
		&guidebookContent, &iconURL)
	if err != nil {
		return nil, err
	}

	unit.ID, _ = uuid.Parse(unitID)
	unit.CourseID, _ = uuid.Parse(courseID)
	unit.GuidebookContent = guidebookContent.String
	unit.IconURL = iconURL.String

	skills, err := s.getSkillsForUnit(ctx, unit.ID)
	if err != nil {
		return nil, err
	}
	unit.Skills = skills

	return &unit, nil
}

// GetSkill returns a skill by ID with its lessons
func (s *CourseStore) GetSkill(ctx context.Context, id uuid.UUID) (*store.Skill, error) {
	var skill store.Skill
	var skillID, unitID string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, unit_id, position, name, icon_name, levels, lexemes_count
		FROM skills WHERE id = ?
	`, id.String()).Scan(&skillID, &unitID, &skill.Position, &skill.Name, &skill.IconName,
		&skill.Levels, &skill.LexemesCount)
	if err != nil {
		return nil, err
	}

	skill.ID, _ = uuid.Parse(skillID)
	skill.UnitID, _ = uuid.Parse(unitID)

	// Get lessons
	lessons, err := s.getLessonsForSkill(ctx, skill.ID)
	if err != nil {
		return nil, err
	}
	skill.Lessons = lessons

	return &skill, nil
}

func (s *CourseStore) getLessonsForSkill(ctx context.Context, skillID uuid.UUID) ([]store.Lesson, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, skill_id, level, position, exercise_count
		FROM lessons WHERE skill_id = ? ORDER BY level, position
	`, skillID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lessons []store.Lesson
	for rows.Next() {
		var lesson store.Lesson
		var id, sID string
		if err := rows.Scan(&id, &sID, &lesson.Level, &lesson.Position, &lesson.ExerciseCount); err != nil {
			return nil, err
		}
		lesson.ID, _ = uuid.Parse(id)
		lesson.SkillID, _ = uuid.Parse(sID)
		lessons = append(lessons, lesson)
	}

	return lessons, rows.Err()
}

// GetLesson returns a lesson by ID
func (s *CourseStore) GetLesson(ctx context.Context, id uuid.UUID) (*store.Lesson, error) {
	var lesson store.Lesson
	var lessonID, skillID string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, skill_id, level, position, exercise_count
		FROM lessons WHERE id = ?
	`, id.String()).Scan(&lessonID, &skillID, &lesson.Level, &lesson.Position, &lesson.ExerciseCount)
	if err != nil {
		return nil, err
	}

	lesson.ID, _ = uuid.Parse(lessonID)
	lesson.SkillID, _ = uuid.Parse(skillID)

	return &lesson, nil
}

// GetExercises returns exercises for a lesson
func (s *CourseStore) GetExercises(ctx context.Context, lessonID uuid.UUID) ([]store.Exercise, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, lesson_id, type, prompt, correct_answer, choices, audio_url, image_url, hints, difficulty
		FROM exercises WHERE lesson_id = ?
	`, lessonID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []store.Exercise
	for rows.Next() {
		var ex store.Exercise
		var id, lID string
		var choicesJSON, hintsJSON, audioURL, imageURL sql.NullString

		if err := rows.Scan(&id, &lID, &ex.Type, &ex.Prompt, &ex.CorrectAnswer, &choicesJSON,
			&audioURL, &imageURL, &hintsJSON, &ex.Difficulty); err != nil {
			return nil, err
		}

		ex.ID, _ = uuid.Parse(id)
		ex.LessonID, _ = uuid.Parse(lID)
		ex.AudioURL = audioURL.String
		ex.ImageURL = imageURL.String

		if choicesJSON.Valid {
			_ = json.Unmarshal([]byte(choicesJSON.String), &ex.Choices)
		}
		if hintsJSON.Valid {
			_ = json.Unmarshal([]byte(hintsJSON.String), &ex.Hints)
		}

		exercises = append(exercises, ex)
	}

	return exercises, rows.Err()
}

// GetStories returns stories for a course
func (s *CourseStore) GetStories(ctx context.Context, courseID uuid.UUID) ([]store.Story, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, course_id, title, difficulty, character_ids, content, xp_reward
		FROM stories WHERE course_id = ? ORDER BY difficulty
	`, courseID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stories []store.Story
	for rows.Next() {
		var story store.Story
		var id, cID string
		var charIDsJSON, contentJSON sql.NullString

		if err := rows.Scan(&id, &cID, &story.Title, &story.Difficulty, &charIDsJSON, &contentJSON, &story.XPReward); err != nil {
			return nil, err
		}

		story.ID, _ = uuid.Parse(id)
		story.CourseID, _ = uuid.Parse(cID)

		if charIDsJSON.Valid {
			_ = json.Unmarshal([]byte(charIDsJSON.String), &story.CharacterIDs)
		}
		if contentJSON.Valid {
			_ = json.Unmarshal([]byte(contentJSON.String), &story.Content)
		}

		stories = append(stories, story)
	}

	return stories, rows.Err()
}

// GetStory returns a story by ID
func (s *CourseStore) GetStory(ctx context.Context, id uuid.UUID) (*store.Story, error) {
	var story store.Story
	var storyID, courseID string
	var charIDsJSON, contentJSON sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, course_id, title, difficulty, character_ids, content, xp_reward
		FROM stories WHERE id = ?
	`, id.String()).Scan(&storyID, &courseID, &story.Title, &story.Difficulty, &charIDsJSON, &contentJSON, &story.XPReward)
	if err != nil {
		return nil, err
	}

	story.ID, _ = uuid.Parse(storyID)
	story.CourseID, _ = uuid.Parse(courseID)

	if charIDsJSON.Valid {
		_ = json.Unmarshal([]byte(charIDsJSON.String), &story.CharacterIDs)
	}
	if contentJSON.Valid {
		_ = json.Unmarshal([]byte(contentJSON.String), &story.Content)
	}

	return &story, nil
}
