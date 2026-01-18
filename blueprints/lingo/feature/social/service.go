package social

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrCannotFollowSelf     = errors.New("cannot follow yourself")
	ErrAlreadyFollowing     = errors.New("already following this user")
	ErrNotFollowing         = errors.New("not following this user")
	ErrNotificationNotFound = errors.New("notification not found")
)

// Service handles social business logic
type Service struct {
	store  store.Store
	social store.SocialStore
	users  store.UserStore
}

// NewService creates a new social service
func NewService(st store.Store) *Service {
	return &Service{
		store:  st,
		social: st.Social(),
		users:  st.Users(),
	}
}

// FriendInfo represents a friend with additional info
type FriendInfo struct {
	User        *store.User `json:"user"`
	IsFollowing bool        `json:"is_following"`
	IsFollower  bool        `json:"is_follower"`
	FriendStreak int        `json:"friend_streak"`
}

// Follow follows a user
func (s *Service) Follow(ctx context.Context, followerID, followingID uuid.UUID) error {
	if followerID == followingID {
		return ErrCannotFollowSelf
	}

	// Verify target user exists
	_, err := s.users.GetByID(ctx, followingID)
	if err != nil {
		return ErrUserNotFound
	}

	return s.social.Follow(ctx, followerID, followingID)
}

// Unfollow unfollows a user
func (s *Service) Unfollow(ctx context.Context, followerID, followingID uuid.UUID) error {
	return s.social.Unfollow(ctx, followerID, followingID)
}

// GetFollowers returns users following the given user
func (s *Service) GetFollowers(ctx context.Context, userID uuid.UUID) ([]store.User, error) {
	return s.social.GetFollowers(ctx, userID)
}

// GetFollowing returns users the given user is following
func (s *Service) GetFollowing(ctx context.Context, userID uuid.UUID) ([]store.User, error) {
	return s.social.GetFollowing(ctx, userID)
}

// GetFriends returns mutual follows (friends)
func (s *Service) GetFriends(ctx context.Context, userID uuid.UUID) ([]FriendInfo, error) {
	following, _ := s.social.GetFollowing(ctx, userID)
	followers, _ := s.social.GetFollowers(ctx, userID)

	// Create a map of followers for quick lookup
	followerMap := make(map[uuid.UUID]bool)
	for _, f := range followers {
		followerMap[f.ID] = true
	}

	// Find mutual follows
	var friends []FriendInfo
	for _, f := range following {
		isMutual := followerMap[f.ID]

		// Get friend streak
		streaks, _ := s.social.GetFriendStreaks(ctx, userID)
		var friendStreak int
		for _, streak := range streaks {
			if streak.User1ID == f.ID || streak.User2ID == f.ID {
				friendStreak = streak.StreakDays
				break
			}
		}

		friends = append(friends, FriendInfo{
			User:         &f,
			IsFollowing:  true,
			IsFollower:   isMutual,
			FriendStreak: friendStreak,
		})
	}

	return friends, nil
}

// GetFriendLeaderboard returns XP leaderboard among friends
func (s *Service) GetFriendLeaderboard(ctx context.Context, userID uuid.UUID) ([]store.User, error) {
	return s.social.GetFriendLeaderboard(ctx, userID)
}

// GetFriendQuests returns active friend quests
func (s *Service) GetFriendQuests(ctx context.Context, userID uuid.UUID) ([]store.FriendQuest, error) {
	return s.social.GetFriendQuests(ctx, userID)
}

// CreateFriendQuest creates a new friend quest
func (s *Service) CreateFriendQuest(ctx context.Context, user1ID, user2ID uuid.UUID, questType string, target int) (*store.FriendQuest, error) {
	quest := &store.FriendQuest{
		ID:          uuid.New(),
		User1ID:     user1ID,
		User2ID:     user2ID,
		QuestType:   questType,
		TargetValue: target,
		StartsAt:    time.Now(),
		EndsAt:      time.Now().Add(7 * 24 * time.Hour), // 1 week
	}

	if err := s.social.CreateFriendQuest(ctx, quest); err != nil {
		return nil, err
	}

	return quest, nil
}

// UpdateFriendQuestProgress updates progress on a friend quest
func (s *Service) UpdateFriendQuestProgress(ctx context.Context, questID, userID uuid.UUID, progress int) error {
	quests, _ := s.social.GetFriendQuests(ctx, userID)

	for _, quest := range quests {
		if quest.ID == questID {
			if quest.User1ID == userID {
				quest.User1Progress = progress
			} else if quest.User2ID == userID {
				quest.User2Progress = progress
			}

			// Check if quest is completed
			if quest.User1Progress >= quest.TargetValue && quest.User2Progress >= quest.TargetValue {
				quest.Completed = true
			}

			return s.social.UpdateFriendQuest(ctx, &quest)
		}
	}

	return nil
}

// GetFriendStreaks returns friend streaks
func (s *Service) GetFriendStreaks(ctx context.Context, userID uuid.UUID) ([]store.FriendStreak, error) {
	return s.social.GetFriendStreaks(ctx, userID)
}

// UpdateFriendStreak updates a friend streak
func (s *Service) UpdateFriendStreak(ctx context.Context, user1ID, user2ID uuid.UUID) error {
	streaks, _ := s.social.GetFriendStreaks(ctx, user1ID)

	for _, streak := range streaks {
		if (streak.User1ID == user1ID && streak.User2ID == user2ID) ||
			(streak.User1ID == user2ID && streak.User2ID == user1ID) {

			// Check if both were active today
			today := time.Now().Truncate(24 * time.Hour)
			if streak.LastBothActive.Truncate(24 * time.Hour).Equal(today) {
				// Already updated today
				return nil
			}

			// Check if yesterday to maintain streak
			yesterday := today.AddDate(0, 0, -1)
			if streak.LastBothActive.Truncate(24 * time.Hour).Equal(yesterday) {
				streak.StreakDays++
			} else {
				// Streak broken, reset
				streak.StreakDays = 1
			}
			streak.LastBothActive = today

			return s.social.UpdateFriendStreak(ctx, &streak)
		}
	}

	// Create new friend streak
	newStreak := &store.FriendStreak{
		ID:             uuid.New(),
		User1ID:        user1ID,
		User2ID:        user2ID,
		StreakDays:     1,
		StartedAt:      time.Now(),
		LastBothActive: time.Now(),
	}

	return s.social.UpdateFriendStreak(ctx, newStreak)
}

// GetNotifications returns notifications for a user
func (s *Service) GetNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool) ([]store.Notification, error) {
	return s.social.GetNotifications(ctx, userID, unreadOnly)
}

// MarkNotificationRead marks a notification as read
func (s *Service) MarkNotificationRead(ctx context.Context, notificationID uuid.UUID) error {
	return s.social.MarkNotificationRead(ctx, notificationID)
}

// CreateNotification creates a new notification
func (s *Service) CreateNotification(ctx context.Context, userID uuid.UUID, notifType, title, body string, data map[string]any) error {
	notif := &store.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Body:      body,
		Data:      data,
		Read:      false,
		CreatedAt: time.Now(),
	}

	return s.social.CreateNotification(ctx, notif)
}
