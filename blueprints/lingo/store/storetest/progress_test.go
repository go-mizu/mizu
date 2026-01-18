package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func createTestUser(t *testing.T, s store.Store) *store.User {
	t.Helper()
	ctx := context.Background()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	user := &store.User{
		ID:                uuid.New(),
		Email:             uuid.New().String() + "@example.com",
		Username:          "user" + uuid.New().String()[:8],
		DisplayName:       "Test User",
		EncryptedPassword: string(hashedPassword),
		Gems:              500,
		Hearts:            5,
		DailyGoalMinutes:  10,
		CreatedAt:         time.Now(),
	}

	if err := s.Users().Create(ctx, user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

func TestProgressStore_EnrollCourse(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		user := createTestUser(t, s)
		progressStore := s.Progress()

		// Get a course
		courses, _ := s.Courses().ListCourses(ctx, "en")
		if len(courses) < 1 {
			t.Skip("no courses available")
		}
		courseID := courses[0].ID

		// Enroll
		err := progressStore.EnrollCourse(ctx, user.ID, courseID)
		assertNoError(t, err, "enroll course")

		// Verify enrollment
		userCourses, err := progressStore.GetUserCourses(ctx, user.ID)
		assertNoError(t, err, "get user courses")
		assertEqual(t, 1, len(userCourses), "enrolled courses count")
		assertEqual(t, courseID, userCourses[0].CourseID, "course id")
	})
}

func TestProgressStore_GetUserCourse(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		user := createTestUser(t, s)
		progressStore := s.Progress()

		// Get a course
		courses, _ := s.Courses().ListCourses(ctx, "en")
		if len(courses) < 1 {
			t.Skip("no courses available")
		}
		courseID := courses[0].ID

		// Enroll
		_ = progressStore.EnrollCourse(ctx, user.ID, courseID)

		// Get specific enrollment
		userCourse, err := progressStore.GetUserCourse(ctx, user.ID, courseID)
		assertNoError(t, err, "get user course")
		assertEqual(t, courseID, userCourse.CourseID, "course id")
		assertEqual(t, int64(0), userCourse.XPEarned, "initial xp")

		// Test non-enrolled course
		_, err = progressStore.GetUserCourse(ctx, user.ID, uuid.New())
		assertError(t, err, "get non-enrolled course")
	})
}

func TestProgressStore_UpdateUserCourse(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		user := createTestUser(t, s)
		progressStore := s.Progress()

		// Get a course
		courses, _ := s.Courses().ListCourses(ctx, "en")
		if len(courses) < 1 {
			t.Skip("no courses available")
		}
		courseID := courses[0].ID

		// Enroll
		_ = progressStore.EnrollCourse(ctx, user.ID, courseID)

		// Get and update
		userCourse, _ := progressStore.GetUserCourse(ctx, user.ID, courseID)
		userCourse.XPEarned = 100
		userCourse.CrownsEarned = 3
		now := time.Now()
		userCourse.LastPracticedAt = &now

		err := progressStore.UpdateUserCourse(ctx, userCourse)
		assertNoError(t, err, "update user course")

		// Verify update
		updated, _ := progressStore.GetUserCourse(ctx, user.ID, courseID)
		assertEqual(t, int64(100), updated.XPEarned, "xp earned")
		assertEqual(t, 3, updated.CrownsEarned, "crowns earned")
	})
}

func TestProgressStore_RecordMistake(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		user := createTestUser(t, s)
		progressStore := s.Progress()

		// Get an exercise
		courses, _ := s.Courses().ListCourses(ctx, "en")
		if len(courses) < 1 {
			t.Skip("no courses available")
		}
		path, _ := s.Courses().GetCoursePath(ctx, courses[0].ID)
		if len(path) < 1 || len(path[0].Skills) < 1 {
			t.Skip("no skills available")
		}
		skill, _ := s.Courses().GetSkill(ctx, path[0].Skills[0].ID)
		if len(skill.Lessons) < 1 {
			t.Skip("no lessons available")
		}
		exercises, _ := s.Courses().GetExercises(ctx, skill.Lessons[0].ID)
		if len(exercises) < 1 {
			t.Skip("no exercises available")
		}

		// Record mistake
		mistake := &store.UserMistake{
			ID:            uuid.New(),
			UserID:        user.ID,
			ExerciseID:    exercises[0].ID,
			UserAnswer:    "wrong",
			CorrectAnswer: exercises[0].CorrectAnswer,
			MistakeType:   "incorrect",
			CreatedAt:     time.Now(),
		}

		err := progressStore.RecordMistake(ctx, mistake)
		assertNoError(t, err, "record mistake")

		// Get mistakes
		mistakes, err := progressStore.GetUserMistakes(ctx, user.ID, 10)
		assertNoError(t, err, "get mistakes")
		assertEqual(t, 1, len(mistakes), "mistakes count")
	})
}

func TestProgressStore_LessonSession(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		user := createTestUser(t, s)
		progressStore := s.Progress()

		// Get a lesson
		courses, _ := s.Courses().ListCourses(ctx, "en")
		if len(courses) < 1 {
			t.Skip("no courses available")
		}
		path, _ := s.Courses().GetCoursePath(ctx, courses[0].ID)
		if len(path) < 1 || len(path[0].Skills) < 1 {
			t.Skip("no skills available")
		}
		skill, _ := s.Courses().GetSkill(ctx, path[0].Skills[0].ID)
		if len(skill.Lessons) < 1 {
			t.Skip("no lessons available")
		}
		lessonID := skill.Lessons[0].ID

		// Start session
		session := &store.LessonSession{
			ID:        uuid.New(),
			UserID:    user.ID,
			LessonID:  lessonID,
			StartedAt: time.Now(),
		}

		err := progressStore.StartLessonSession(ctx, session)
		assertNoError(t, err, "start lesson session")

		// Complete session
		now := time.Now()
		session.CompletedAt = &now
		session.XPEarned = 15
		session.MistakesCount = 2
		session.IsPerfect = false

		err = progressStore.CompleteLessonSession(ctx, session)
		assertNoError(t, err, "complete lesson session")
	})
}

func TestProgressStore_XPEvents(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		user := createTestUser(t, s)
		progressStore := s.Progress()

		// Record XP event
		event := &store.XPEvent{
			ID:        uuid.New(),
			UserID:    user.ID,
			Amount:    15,
			Source:    "lesson",
			CreatedAt: time.Now(),
		}

		err := progressStore.RecordXPEvent(ctx, event)
		assertNoError(t, err, "record xp event")

		// Get XP history
		history, err := progressStore.GetXPHistory(ctx, user.ID, 7)
		assertNoError(t, err, "get xp history")
		assertEqual(t, 1, len(history), "history count")
		assertEqual(t, 15, history[0].Amount, "xp amount")
	})
}

func TestProgressStore_StreakHistory(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		user := createTestUser(t, s)
		progressStore := s.Progress()

		// Record streak day
		err := progressStore.RecordStreakDay(ctx, user.ID, 50, 3, 600)
		assertNoError(t, err, "record streak day")

		// Get streak history
		history, err := progressStore.GetStreakHistory(ctx, user.ID, 30)
		assertNoError(t, err, "get streak history")
		assertEqual(t, 1, len(history), "history count")
		assertEqual(t, 50, history[0].XPEarned, "xp earned")
		assertEqual(t, 3, history[0].LessonsCompleted, "lessons completed")
	})
}
