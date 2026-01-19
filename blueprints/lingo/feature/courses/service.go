package courses

import (
	"context"
	"errors"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

var (
	ErrCourseNotFound  = errors.New("course not found")
	ErrAlreadyEnrolled = errors.New("already enrolled in this course")
	ErrUnitNotFound    = errors.New("unit not found")
	ErrSkillNotFound   = errors.New("skill not found")
	ErrStoryNotFound   = errors.New("story not found")
)

// Service handles course business logic
type Service struct {
	store    store.Store
	courses  store.CourseStore
	progress store.ProgressStore
	stories  store.StoryStore
}

// NewService creates a new course service
func NewService(st store.Store) *Service {
	return &Service{
		store:    st,
		courses:  st.Courses(),
		progress: st.Progress(),
		stories:  st.Stories(),
	}
}

// ListLanguages returns all available languages
func (s *Service) ListLanguages(ctx context.Context) ([]store.Language, error) {
	return s.courses.ListLanguages(ctx)
}

// ListCourses returns courses available for a source language
func (s *Service) ListCourses(ctx context.Context, fromLang string) ([]store.Course, error) {
	return s.courses.ListCourses(ctx, fromLang)
}

// GetCourse returns a course by ID
func (s *Service) GetCourse(ctx context.Context, id uuid.UUID) (*store.Course, error) {
	course, err := s.courses.GetCourse(ctx, id)
	if err != nil {
		return nil, ErrCourseNotFound
	}
	return course, nil
}

// GetCoursePath returns the full learning path for a course
func (s *Service) GetCoursePath(ctx context.Context, courseID uuid.UUID) ([]store.Unit, error) {
	return s.courses.GetCoursePath(ctx, courseID)
}

// EnrollInCourse enrolls a user in a course
func (s *Service) EnrollInCourse(ctx context.Context, userID, courseID uuid.UUID) error {
	// Check if already enrolled
	existing, _ := s.progress.GetUserCourse(ctx, userID, courseID)
	if existing != nil {
		return ErrAlreadyEnrolled
	}

	// Verify course exists
	_, err := s.courses.GetCourse(ctx, courseID)
	if err != nil {
		return ErrCourseNotFound
	}

	return s.progress.EnrollCourse(ctx, userID, courseID)
}

// GetUserCourses returns all courses a user is enrolled in
func (s *Service) GetUserCourses(ctx context.Context, userID uuid.UUID) ([]store.UserCourse, error) {
	return s.progress.GetUserCourses(ctx, userID)
}

// GetUnit returns a unit by ID with its skills
func (s *Service) GetUnit(ctx context.Context, id uuid.UUID) (*store.Unit, error) {
	unit, err := s.courses.GetUnit(ctx, id)
	if err != nil {
		return nil, ErrUnitNotFound
	}
	return unit, nil
}

// GetSkill returns a skill by ID with its lessons
func (s *Service) GetSkill(ctx context.Context, id uuid.UUID) (*store.Skill, error) {
	skill, err := s.courses.GetSkill(ctx, id)
	if err != nil {
		return nil, ErrSkillNotFound
	}
	return skill, nil
}

// GetUserSkillProgress returns user's progress on a skill
func (s *Service) GetUserSkillProgress(ctx context.Context, userID, skillID uuid.UUID) (*store.UserSkill, error) {
	return s.progress.GetUserSkill(ctx, userID, skillID)
}

// GetStories returns stories for a course
func (s *Service) GetStories(ctx context.Context, courseID uuid.UUID) ([]store.Story, error) {
	return s.courses.GetStories(ctx, courseID)
}

// GetStory returns a story by ID with characters and elements
func (s *Service) GetStory(ctx context.Context, id uuid.UUID) (*store.Story, error) {
	story, err := s.stories.GetStory(ctx, id)
	if err != nil {
		return nil, ErrStoryNotFound
	}
	return story, nil
}

// GetLexemesByCourse returns all vocabulary for a course
func (s *Service) GetLexemesByCourse(ctx context.Context, courseID uuid.UUID) ([]store.Lexeme, error) {
	return s.courses.GetLexemesByCourse(ctx, courseID)
}

// CourseWithProgress represents a course with user progress
type CourseWithProgress struct {
	Course   store.Course     `json:"course"`
	Progress *store.UserCourse `json:"progress,omitempty"`
}

// GetCourseWithProgress returns a course with user's progress
func (s *Service) GetCourseWithProgress(ctx context.Context, userID, courseID uuid.UUID) (*CourseWithProgress, error) {
	course, err := s.courses.GetCourse(ctx, courseID)
	if err != nil {
		return nil, ErrCourseNotFound
	}

	progress, _ := s.progress.GetUserCourse(ctx, userID, courseID)

	return &CourseWithProgress{
		Course:   *course,
		Progress: progress,
	}, nil
}
