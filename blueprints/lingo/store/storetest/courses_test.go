package storetest

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

func TestCourseStore_ListLanguages(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		courses := s.Courses()

		languages, err := courses.ListLanguages(ctx)
		assertNoError(t, err, "list languages")

		if len(languages) < 1 {
			t.Fatal("expected at least 1 language")
		}

		// Verify language fields
		found := false
		for _, lang := range languages {
			if lang.ID == "en" {
				found = true
				assertEqual(t, "English", lang.Name, "language name")
				assertEqual(t, true, lang.Enabled, "language enabled")
			}
		}

		if !found {
			t.Fatal("expected to find English language")
		}
	})
}

func TestCourseStore_ListCourses(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		courseStore := s.Courses()

		// List courses from English
		courses, err := courseStore.ListCourses(ctx, "en")
		assertNoError(t, err, "list courses")

		if len(courses) < 1 {
			t.Fatal("expected at least 1 course")
		}

		// Verify course fields
		for _, course := range courses {
			assertEqual(t, "en", course.FromLanguageID, "from language")
			if course.Title == "" {
				t.Fatal("expected course title")
			}
		}
	})
}

func TestCourseStore_GetCourse(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		courseStore := s.Courses()

		// Get list first
		courses, _ := courseStore.ListCourses(ctx, "en")
		if len(courses) < 1 {
			t.Skip("no courses available")
		}

		courseID := courses[0].ID

		// Get specific course
		course, err := courseStore.GetCourse(ctx, courseID)
		assertNoError(t, err, "get course")
		assertEqual(t, courseID, course.ID, "course id")

		// Test non-existent course
		_, err = courseStore.GetCourse(ctx, uuid.New())
		assertError(t, err, "get non-existent course")
	})
}

func TestCourseStore_GetCoursePath(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		courseStore := s.Courses()

		// Get list first
		courses, _ := courseStore.ListCourses(ctx, "en")
		if len(courses) < 1 {
			t.Skip("no courses available")
		}

		courseID := courses[0].ID

		// Get course path
		path, err := courseStore.GetCoursePath(ctx, courseID)
		assertNoError(t, err, "get course path")

		if len(path) < 1 {
			t.Fatal("expected at least 1 unit in path")
		}

		// Verify unit structure
		for _, unit := range path {
			if unit.Title == "" {
				t.Fatal("expected unit title")
			}
			if len(unit.Skills) < 1 {
				t.Fatal("expected at least 1 skill in unit")
			}
		}
	})
}

func TestCourseStore_GetUnit(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		courseStore := s.Courses()

		// Get course path first
		courses, _ := courseStore.ListCourses(ctx, "en")
		if len(courses) < 1 {
			t.Skip("no courses available")
		}

		path, _ := courseStore.GetCoursePath(ctx, courses[0].ID)
		if len(path) < 1 {
			t.Skip("no units available")
		}

		unitID := path[0].ID

		// Get unit
		unit, err := courseStore.GetUnit(ctx, unitID)
		assertNoError(t, err, "get unit")
		assertEqual(t, unitID, unit.ID, "unit id")

		if len(unit.Skills) < 1 {
			t.Fatal("expected at least 1 skill")
		}
	})
}

func TestCourseStore_GetSkill(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		courseStore := s.Courses()

		// Get course path first
		courses, _ := courseStore.ListCourses(ctx, "en")
		if len(courses) < 1 {
			t.Skip("no courses available")
		}

		path, _ := courseStore.GetCoursePath(ctx, courses[0].ID)
		if len(path) < 1 || len(path[0].Skills) < 1 {
			t.Skip("no skills available")
		}

		skillID := path[0].Skills[0].ID

		// Get skill
		skill, err := courseStore.GetSkill(ctx, skillID)
		assertNoError(t, err, "get skill")
		assertEqual(t, skillID, skill.ID, "skill id")

		if len(skill.Lessons) < 1 {
			t.Fatal("expected at least 1 lesson")
		}
	})
}

func TestCourseStore_GetLesson(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		courseStore := s.Courses()

		// Get course path first
		courses, _ := courseStore.ListCourses(ctx, "en")
		if len(courses) < 1 {
			t.Skip("no courses available")
		}

		path, _ := courseStore.GetCoursePath(ctx, courses[0].ID)
		if len(path) < 1 || len(path[0].Skills) < 1 {
			t.Skip("no skills available")
		}

		skill, _ := courseStore.GetSkill(ctx, path[0].Skills[0].ID)
		if len(skill.Lessons) < 1 {
			t.Skip("no lessons available")
		}

		lessonID := skill.Lessons[0].ID

		// Get lesson
		lesson, err := courseStore.GetLesson(ctx, lessonID)
		assertNoError(t, err, "get lesson")
		assertEqual(t, lessonID, lesson.ID, "lesson id")
	})
}

func TestCourseStore_GetExercises(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		courseStore := s.Courses()

		// Get course path first
		courses, _ := courseStore.ListCourses(ctx, "en")
		if len(courses) < 1 {
			t.Skip("no courses available")
		}

		path, _ := courseStore.GetCoursePath(ctx, courses[0].ID)
		if len(path) < 1 || len(path[0].Skills) < 1 {
			t.Skip("no skills available")
		}

		skill, _ := courseStore.GetSkill(ctx, path[0].Skills[0].ID)
		if len(skill.Lessons) < 1 {
			t.Skip("no lessons available")
		}

		lessonID := skill.Lessons[0].ID

		// Get exercises
		exercises, err := courseStore.GetExercises(ctx, lessonID)
		assertNoError(t, err, "get exercises")

		if len(exercises) < 1 {
			t.Fatal("expected at least 1 exercise")
		}

		// Verify exercise fields
		for _, ex := range exercises {
			if ex.Type == "" {
				t.Fatal("expected exercise type")
			}
			if ex.Prompt == "" {
				t.Fatal("expected exercise prompt")
			}
		}
	})
}
