package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

func TestSocialStore_Follow(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		user1 := createTestUser(t, s)
		user2 := createTestUser(t, s)
		social := s.Social()

		// Follow
		err := social.Follow(ctx, user1.ID, user2.ID)
		assertNoError(t, err, "follow user")

		// Verify following
		following, err := social.GetFollowing(ctx, user1.ID)
		assertNoError(t, err, "get following")
		assertEqual(t, 1, len(following), "following count")
		assertEqual(t, user2.ID, following[0].ID, "following user id")

		// Verify followers
		followers, err := social.GetFollowers(ctx, user2.ID)
		assertNoError(t, err, "get followers")
		assertEqual(t, 1, len(followers), "followers count")
		assertEqual(t, user1.ID, followers[0].ID, "follower user id")
	})
}

func TestSocialStore_Unfollow(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		user1 := createTestUser(t, s)
		user2 := createTestUser(t, s)
		social := s.Social()

		// Follow then unfollow
		_ = social.Follow(ctx, user1.ID, user2.ID)

		err := social.Unfollow(ctx, user1.ID, user2.ID)
		assertNoError(t, err, "unfollow user")

		// Verify unfollowed
		following, _ := social.GetFollowing(ctx, user1.ID)
		assertEqual(t, 0, len(following), "following count after unfollow")
	})
}

func TestSocialStore_GetFriendLeaderboard(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		user1 := createTestUser(t, s)
		user2 := createTestUser(t, s)
		user3 := createTestUser(t, s)
		social := s.Social()

		// user1 follows user2 and user3
		_ = social.Follow(ctx, user1.ID, user2.ID)
		_ = social.Follow(ctx, user1.ID, user3.ID)

		// Get friend leaderboard
		leaderboard, err := social.GetFriendLeaderboard(ctx, user1.ID)
		assertNoError(t, err, "get friend leaderboard")

		// Should include user1's friends
		if len(leaderboard) < 2 {
			t.Fatalf("expected at least 2 users in leaderboard, got %d", len(leaderboard))
		}
	})
}

func TestSocialStore_FriendQuest(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		user1 := createTestUser(t, s)
		user2 := createTestUser(t, s)
		social := s.Social()

		// Create friend quest
		quest := &store.FriendQuest{
			ID:            uuid.New(),
			User1ID:       user1.ID,
			User2ID:       user2.ID,
			QuestType:     "complete_lessons",
			TargetValue:   5,
			User1Progress: 0,
			User2Progress: 0,
			StartsAt:      time.Now(),
			EndsAt:        time.Now().Add(24 * time.Hour),
			Completed:     false,
		}

		err := social.CreateFriendQuest(ctx, quest)
		assertNoError(t, err, "create friend quest")

		// Get user's friend quests
		quests, err := social.GetFriendQuests(ctx, user1.ID)
		assertNoError(t, err, "get friend quests")
		assertEqual(t, 1, len(quests), "quests count")

		// Update progress
		quest.User1Progress = 3
		quest.User2Progress = 2
		err = social.UpdateFriendQuest(ctx, quest)
		assertNoError(t, err, "update friend quest progress")

		// Complete quest
		quest.User1Progress = 5
		quest.User2Progress = 5
		quest.Completed = true
		err = social.UpdateFriendQuest(ctx, quest)
		assertNoError(t, err, "complete friend quest")

		// Verify completion
		quests, _ = social.GetFriendQuests(ctx, user1.ID)
		assertEqual(t, true, quests[0].Completed, "quest completed")
	})
}

func TestSocialStore_FriendStreak(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		user1 := createTestUser(t, s)
		user2 := createTestUser(t, s)
		social := s.Social()

		// Users must follow each other to have friend streaks
		_ = social.Follow(ctx, user1.ID, user2.ID)
		_ = social.Follow(ctx, user2.ID, user1.ID)

		// Get friend streaks
		streaks, err := social.GetFriendStreaks(ctx, user1.ID)
		assertNoError(t, err, "get friend streaks")

		// Update friend streak if any exist
		if len(streaks) > 0 {
			streak := &streaks[0]
			streak.StreakDays = 5
			streak.LastBothActive = time.Now()

			err = social.UpdateFriendStreak(ctx, streak)
			assertNoError(t, err, "update friend streak")
		}
	})
}

func TestSocialStore_Notifications(t *testing.T) {
	ts := setupTestStores(t)

	ts.forEachStore(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		user := createTestUser(t, s)
		social := s.Social()

		// Create notifications
		notification1 := &store.Notification{
			ID:        uuid.New(),
			UserID:    user.ID,
			Type:      "achievement",
			Title:     "Achievement Unlocked!",
			Body:      "You earned the Champion badge",
			Read:      false,
			CreatedAt: time.Now(),
		}
		notification2 := &store.Notification{
			ID:        uuid.New(),
			UserID:    user.ID,
			Type:      "friend_request",
			Title:     "New Friend",
			Body:      "Someone started following you",
			Read:      false,
			CreatedAt: time.Now(),
		}

		err := social.CreateNotification(ctx, notification1)
		assertNoError(t, err, "create notification 1")
		err = social.CreateNotification(ctx, notification2)
		assertNoError(t, err, "create notification 2")

		// Get all notifications
		notifications, err := social.GetNotifications(ctx, user.ID, false)
		assertNoError(t, err, "get notifications")
		assertEqual(t, 2, len(notifications), "notifications count")

		// Get unread only
		unreadNotifications, err := social.GetNotifications(ctx, user.ID, true)
		assertNoError(t, err, "get unread notifications")
		assertEqual(t, 2, len(unreadNotifications), "unread notifications count")

		// Mark one as read
		err = social.MarkNotificationRead(ctx, notification1.ID)
		assertNoError(t, err, "mark notification read")

		// Verify unread count changed
		unreadNotifications, _ = social.GetNotifications(ctx, user.ID, true)
		assertEqual(t, 1, len(unreadNotifications), "unread count after marking one read")
	})
}
