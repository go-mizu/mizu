package storetest

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
)

func TestAchievementStore_GetAchievements(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		achievements := s.Achievements()

		// Get all achievements
		allAchievements, err := achievements.GetAchievements(ctx)
		assertNoError(t, err, "get achievements")

		if len(allAchievements) < 1 {
			t.Fatal("expected at least 1 achievement")
		}

		// Verify achievement fields
		for _, ach := range allAchievements {
			if ach.Name == "" {
				t.Fatal("expected achievement name")
			}
			if ach.Category == "" {
				t.Fatal("expected achievement category")
			}
			if ach.ID == "" {
				t.Fatal("expected achievement id")
			}
		}
	})
}

func TestAchievementStore_GetUserAchievements(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		user := createTestUser(t, s)
		achievementStore := s.Achievements()

		// Initially no achievements
		userAchievements, err := achievementStore.GetUserAchievements(ctx, user.ID)
		assertNoError(t, err, "get user achievements")
		assertEqual(t, 0, len(userAchievements), "initial user achievements count")
	})
}

func TestAchievementStore_CheckAndUnlock(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		user := createTestUser(t, s)
		achievementStore := s.Achievements()

		// Get achievements
		allAchievements, _ := achievementStore.GetAchievements(ctx)
		if len(allAchievements) < 1 {
			t.Skip("no achievements available")
		}
		achievementID := allAchievements[0].ID

		// Check and unlock with progress
		ua, err := achievementStore.CheckAndUnlock(ctx, user.ID, achievementID, 10)
		assertNoError(t, err, "check and unlock achievement")

		// Verify user achievement was created/updated
		if ua != nil {
			assertEqual(t, achievementID, ua.AchievementID, "achievement id")
			assertEqual(t, 10, ua.Progress, "progress")
		}

		// Get user achievements to verify
		userAchievements, err := achievementStore.GetUserAchievements(ctx, user.ID)
		assertNoError(t, err, "get user achievements after unlock")

		// Should have at least one achievement tracked
		if len(userAchievements) < 1 {
			t.Fatal("expected at least 1 user achievement after CheckAndUnlock")
		}
	})
}

func TestAchievementStore_UpdateUserAchievement(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		user := createTestUser(t, s)
		achievementStore := s.Achievements()

		// Get achievements
		allAchievements, _ := achievementStore.GetAchievements(ctx)
		if len(allAchievements) < 1 {
			t.Skip("no achievements available")
		}
		achievementID := allAchievements[0].ID

		// Create initial user achievement via CheckAndUnlock
		_, _ = achievementStore.CheckAndUnlock(ctx, user.ID, achievementID, 5)

		// Update with more progress
		ua := &store.UserAchievement{
			UserID:        user.ID,
			AchievementID: achievementID,
			Level:         1,
			Progress:      50,
		}

		err := achievementStore.UpdateUserAchievement(ctx, ua)
		assertNoError(t, err, "update user achievement")

		// Verify update
		userAchievements, _ := achievementStore.GetUserAchievements(ctx, user.ID)
		found := false
		for _, achievement := range userAchievements {
			if achievement.AchievementID == achievementID {
				found = true
				assertEqual(t, 50, achievement.Progress, "updated progress")
				break
			}
		}
		if !found {
			t.Fatal("expected to find updated achievement")
		}
	})
}

func TestAchievementStore_MultipleAchievements(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		user := createTestUser(t, s)
		achievementStore := s.Achievements()

		// Get achievements
		allAchievements, _ := achievementStore.GetAchievements(ctx)
		if len(allAchievements) < 3 {
			t.Skip("need at least 3 achievements")
		}

		// Track progress on multiple achievements
		for i := 0; i < 3; i++ {
			_, err := achievementStore.CheckAndUnlock(ctx, user.ID, allAchievements[i].ID, (i+1)*10)
			assertNoError(t, err, "check and unlock achievement")
		}

		// Verify all are tracked
		userAchievements, err := achievementStore.GetUserAchievements(ctx, user.ID)
		assertNoError(t, err, "get user achievements")
		assertEqual(t, 3, len(userAchievements), "user achievements count")
	})
}

func TestAchievementStore_AchievementLevels(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		user := createTestUser(t, s)
		achievementStore := s.Achievements()

		// Get achievements
		allAchievements, _ := achievementStore.GetAchievements(ctx)
		if len(allAchievements) < 1 {
			t.Skip("no achievements available")
		}

		// Find an achievement with multiple levels
		var multiLevelAch *store.Achievement
		for i := range allAchievements {
			if allAchievements[i].MaxLevel > 1 {
				multiLevelAch = &allAchievements[i]
				break
			}
		}
		if multiLevelAch == nil {
			t.Skip("no multi-level achievements available")
		}

		// Progress through levels
		achievementID := multiLevelAch.ID

		// Level 1
		ua, _ := achievementStore.CheckAndUnlock(ctx, user.ID, achievementID, multiLevelAch.Thresholds[0])
		if ua != nil && ua.Level < 1 {
			t.Fatal("expected level 1 after reaching first threshold")
		}

		// If there's a level 2 threshold, test it
		if len(multiLevelAch.Thresholds) > 1 {
			ua, _ = achievementStore.CheckAndUnlock(ctx, user.ID, achievementID, multiLevelAch.Thresholds[1])
			if ua != nil && ua.Level < 2 {
				t.Fatal("expected level 2 after reaching second threshold")
			}
		}
	})
}

func TestAchievementStore_AchievementCategories(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()
		seedTestData(t, s)

		achievementStore := s.Achievements()

		// Get all achievements
		allAchievements, err := achievementStore.GetAchievements(ctx)
		assertNoError(t, err, "get achievements")

		if len(allAchievements) < 1 {
			t.Skip("no achievements available")
		}

		// Group by category
		categories := make(map[string]int)
		for _, ach := range allAchievements {
			categories[ach.Category]++
		}

		// Should have at least one category
		if len(categories) < 1 {
			t.Fatal("expected at least 1 achievement category")
		}
	})
}
