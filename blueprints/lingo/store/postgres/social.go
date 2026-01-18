package postgres

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SocialStore implements store.SocialStore
type SocialStore struct {
	pool *pgxpool.Pool
}

// Follow follows a user
func (s *SocialStore) Follow(ctx context.Context, followerID, followingID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO follows (follower_id, following_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (follower_id, following_id) DO NOTHING
	`, followerID, followingID)
	if err != nil {
		return fmt.Errorf("follow user: %w", err)
	}
	return nil
}

// Unfollow unfollows a user
func (s *SocialStore) Unfollow(ctx context.Context, followerID, followingID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM follows WHERE follower_id = $1 AND following_id = $2
	`, followerID, followingID)
	if err != nil {
		return fmt.Errorf("unfollow user: %w", err)
	}
	return nil
}

// GetFollowers gets a user's followers
func (s *SocialStore) GetFollowers(ctx context.Context, userID uuid.UUID) ([]store.User, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.email, u.username, u.display_name, u.avatar_url, u.xp_total, u.streak_days
		FROM users u
		JOIN follows f ON u.id = f.follower_id
		WHERE f.following_id = $1
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query followers: %w", err)
	}
	defer rows.Close()

	var users []store.User
	for rows.Next() {
		var u store.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.AvatarURL, &u.XPTotal, &u.StreakDays); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

// GetFollowing gets users a user is following
func (s *SocialStore) GetFollowing(ctx context.Context, userID uuid.UUID) ([]store.User, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.email, u.username, u.display_name, u.avatar_url, u.xp_total, u.streak_days
		FROM users u
		JOIN follows f ON u.id = f.following_id
		WHERE f.follower_id = $1
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query following: %w", err)
	}
	defer rows.Close()

	var users []store.User
	for rows.Next() {
		var u store.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.AvatarURL, &u.XPTotal, &u.StreakDays); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

// GetFriendLeaderboard gets a leaderboard of friends
func (s *SocialStore) GetFriendLeaderboard(ctx context.Context, userID uuid.UUID) ([]store.User, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.email, u.username, u.display_name, u.avatar_url, u.xp_total, u.streak_days
		FROM users u
		WHERE u.id IN (
			SELECT following_id FROM follows WHERE follower_id = $1
			UNION
			SELECT $1
		)
		ORDER BY u.xp_total DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query friend leaderboard: %w", err)
	}
	defer rows.Close()

	var users []store.User
	for rows.Next() {
		var u store.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.AvatarURL, &u.XPTotal, &u.StreakDays); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

// GetFriendQuests gets active friend quests for a user
func (s *SocialStore) GetFriendQuests(ctx context.Context, userID uuid.UUID) ([]store.FriendQuest, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user1_id, user2_id, quest_type, target_value, user1_progress, user2_progress, starts_at, ends_at, completed, rewards_claimed
		FROM friend_quests
		WHERE (user1_id = $1 OR user2_id = $1) AND ends_at > NOW()
		ORDER BY starts_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query friend quests: %w", err)
	}
	defer rows.Close()

	var quests []store.FriendQuest
	for rows.Next() {
		var q store.FriendQuest
		if err := rows.Scan(&q.ID, &q.User1ID, &q.User2ID, &q.QuestType, &q.TargetValue, &q.User1Progress, &q.User2Progress, &q.StartsAt, &q.EndsAt, &q.Completed, &q.RewardsClaimed); err != nil {
			return nil, fmt.Errorf("scan friend quest: %w", err)
		}
		quests = append(quests, q)
	}
	return quests, nil
}

// CreateFriendQuest creates a new friend quest
func (s *SocialStore) CreateFriendQuest(ctx context.Context, quest *store.FriendQuest) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO friend_quests (id, user1_id, user2_id, quest_type, target_value, starts_at, ends_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, quest.ID, quest.User1ID, quest.User2ID, quest.QuestType, quest.TargetValue, quest.StartsAt, quest.EndsAt)
	if err != nil {
		return fmt.Errorf("create friend quest: %w", err)
	}
	return nil
}

// UpdateFriendQuest updates a friend quest
func (s *SocialStore) UpdateFriendQuest(ctx context.Context, quest *store.FriendQuest) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE friend_quests SET user1_progress = $2, user2_progress = $3, completed = $4, rewards_claimed = $5
		WHERE id = $1
	`, quest.ID, quest.User1Progress, quest.User2Progress, quest.Completed, quest.RewardsClaimed)
	if err != nil {
		return fmt.Errorf("update friend quest: %w", err)
	}
	return nil
}

// GetFriendStreaks gets shared streaks with friends
func (s *SocialStore) GetFriendStreaks(ctx context.Context, userID uuid.UUID) ([]store.FriendStreak, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user1_id, user2_id, streak_days, started_at, last_both_active
		FROM friend_streaks
		WHERE user1_id = $1 OR user2_id = $1
		ORDER BY streak_days DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query friend streaks: %w", err)
	}
	defer rows.Close()

	var streaks []store.FriendStreak
	for rows.Next() {
		var fs store.FriendStreak
		if err := rows.Scan(&fs.ID, &fs.User1ID, &fs.User2ID, &fs.StreakDays, &fs.StartedAt, &fs.LastBothActive); err != nil {
			return nil, fmt.Errorf("scan friend streak: %w", err)
		}
		streaks = append(streaks, fs)
	}
	return streaks, nil
}

// UpdateFriendStreak updates a friend streak
func (s *SocialStore) UpdateFriendStreak(ctx context.Context, streak *store.FriendStreak) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO friend_streaks (id, user1_id, user2_id, streak_days, started_at, last_both_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET streak_days = $4, last_both_active = $6
	`, streak.ID, streak.User1ID, streak.User2ID, streak.StreakDays, streak.StartedAt, streak.LastBothActive)
	if err != nil {
		return fmt.Errorf("update friend streak: %w", err)
	}
	return nil
}

// CreateNotification creates a notification
func (s *SocialStore) CreateNotification(ctx context.Context, notif *store.Notification) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notifications (id, user_id, type, title, body, data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, notif.ID, notif.UserID, notif.Type, notif.Title, notif.Body, notif.Data)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return nil
}

// GetNotifications gets notifications for a user
func (s *SocialStore) GetNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool) ([]store.Notification, error) {
	query := `
		SELECT id, user_id, type, title, body, data, read, created_at
		FROM notifications WHERE user_id = $1
	`
	if unreadOnly {
		query += " AND read = false"
	}
	query += " ORDER BY created_at DESC LIMIT 50"

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query notifications: %w", err)
	}
	defer rows.Close()

	var notifications []store.Notification
	for rows.Next() {
		var n store.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Data, &n.Read, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

// MarkNotificationRead marks a notification as read
func (s *SocialStore) MarkNotificationRead(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE notifications SET read = true WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	return nil
}
