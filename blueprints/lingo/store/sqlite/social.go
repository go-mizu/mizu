package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// SocialStore handles social operations
type SocialStore struct {
	db *sql.DB
}

// Follow follows a user
func (s *SocialStore) Follow(ctx context.Context, followerID, followingID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO follows (follower_id, following_id, created_at)
		VALUES (?, ?, ?)
	`, followerID.String(), followingID.String(), time.Now())
	return err
}

// Unfollow unfollows a user
func (s *SocialStore) Unfollow(ctx context.Context, followerID, followingID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM follows WHERE follower_id = ? AND following_id = ?
	`, followerID.String(), followingID.String())
	return err
}

// GetFollowers returns users following the given user
func (s *SocialStore) GetFollowers(ctx context.Context, userID uuid.UUID) ([]store.User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.username, u.display_name, u.avatar_url, u.xp_total, u.streak_days
		FROM users u
		JOIN follows f ON u.id = f.follower_id
		WHERE f.following_id = ?
		ORDER BY f.created_at DESC
	`, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanUsers(rows)
}

// GetFollowing returns users the given user is following
func (s *SocialStore) GetFollowing(ctx context.Context, userID uuid.UUID) ([]store.User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.username, u.display_name, u.avatar_url, u.xp_total, u.streak_days
		FROM users u
		JOIN follows f ON u.id = f.following_id
		WHERE f.follower_id = ?
		ORDER BY f.created_at DESC
	`, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanUsers(rows)
}

// GetFriendLeaderboard returns XP ranking among friends
func (s *SocialStore) GetFriendLeaderboard(ctx context.Context, userID uuid.UUID) ([]store.User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.username, u.display_name, u.avatar_url, u.xp_total, u.streak_days
		FROM users u
		JOIN follows f ON u.id = f.following_id
		WHERE f.follower_id = ?
		UNION
		SELECT id, username, display_name, avatar_url, xp_total, streak_days
		FROM users WHERE id = ?
		ORDER BY xp_total DESC
	`, userID.String(), userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanUsers(rows)
}

func (s *SocialStore) scanUsers(rows *sql.Rows) ([]store.User, error) {
	var users []store.User
	for rows.Next() {
		var user store.User
		var id string

		if err := rows.Scan(&id, &user.Username, &user.DisplayName, &user.AvatarURL, &user.XPTotal, &user.StreakDays); err != nil {
			return nil, err
		}

		user.ID, _ = uuid.Parse(id)
		users = append(users, user)
	}

	return users, rows.Err()
}

// GetFriendQuests returns active friend quests
func (s *SocialStore) GetFriendQuests(ctx context.Context, userID uuid.UUID) ([]store.FriendQuest, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user1_id, user2_id, quest_type, target_value, user1_progress, user2_progress,
			starts_at, ends_at, completed, rewards_claimed
		FROM friend_quests
		WHERE (user1_id = ? OR user2_id = ?) AND ends_at > ?
		ORDER BY ends_at
	`, userID.String(), userID.String(), time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quests []store.FriendQuest
	for rows.Next() {
		var q store.FriendQuest
		var id, u1ID, u2ID string
		var completed, claimed int

		if err := rows.Scan(&id, &u1ID, &u2ID, &q.QuestType, &q.TargetValue, &q.User1Progress,
			&q.User2Progress, &q.StartsAt, &q.EndsAt, &completed, &claimed); err != nil {
			return nil, err
		}

		q.ID, _ = uuid.Parse(id)
		q.User1ID, _ = uuid.Parse(u1ID)
		q.User2ID, _ = uuid.Parse(u2ID)
		q.Completed = completed == 1
		q.RewardsClaimed = claimed == 1

		quests = append(quests, q)
	}

	return quests, rows.Err()
}

// CreateFriendQuest creates a new friend quest
func (s *SocialStore) CreateFriendQuest(ctx context.Context, quest *store.FriendQuest) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO friend_quests (id, user1_id, user2_id, quest_type, target_value, starts_at, ends_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, quest.ID.String(), quest.User1ID.String(), quest.User2ID.String(), quest.QuestType,
		quest.TargetValue, quest.StartsAt, quest.EndsAt)
	return err
}

// UpdateFriendQuest updates a friend quest
func (s *SocialStore) UpdateFriendQuest(ctx context.Context, quest *store.FriendQuest) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE friend_quests SET
			user1_progress = ?, user2_progress = ?, completed = ?, rewards_claimed = ?
		WHERE id = ?
	`, quest.User1Progress, quest.User2Progress, boolToInt(quest.Completed),
		boolToInt(quest.RewardsClaimed), quest.ID.String())
	return err
}

// GetFriendStreaks returns friend streaks for a user
func (s *SocialStore) GetFriendStreaks(ctx context.Context, userID uuid.UUID) ([]store.FriendStreak, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user1_id, user2_id, streak_days, started_at, last_both_active
		FROM friend_streaks
		WHERE user1_id = ? OR user2_id = ?
	`, userID.String(), userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var streaks []store.FriendStreak
	for rows.Next() {
		var fs store.FriendStreak
		var id, u1ID, u2ID string

		if err := rows.Scan(&id, &u1ID, &u2ID, &fs.StreakDays, &fs.StartedAt, &fs.LastBothActive); err != nil {
			return nil, err
		}

		fs.ID, _ = uuid.Parse(id)
		fs.User1ID, _ = uuid.Parse(u1ID)
		fs.User2ID, _ = uuid.Parse(u2ID)

		streaks = append(streaks, fs)
	}

	return streaks, rows.Err()
}

// UpdateFriendStreak updates or creates a friend streak
func (s *SocialStore) UpdateFriendStreak(ctx context.Context, streak *store.FriendStreak) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO friend_streaks (id, user1_id, user2_id, streak_days, started_at, last_both_active)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			streak_days = excluded.streak_days,
			last_both_active = excluded.last_both_active
	`, streak.ID.String(), streak.User1ID.String(), streak.User2ID.String(),
		streak.StreakDays, streak.StartedAt, streak.LastBothActive)
	return err
}

// CreateNotification creates a new notification
func (s *SocialStore) CreateNotification(ctx context.Context, notif *store.Notification) error {
	dataJSON, _ := json.Marshal(notif.Data)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, type, title, body, data, read, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, notif.ID.String(), notif.UserID.String(), notif.Type, notif.Title, notif.Body,
		string(dataJSON), boolToInt(notif.Read), notif.CreatedAt)
	return err
}

// GetNotifications returns notifications for a user
func (s *SocialStore) GetNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool) ([]store.Notification, error) {
	query := `
		SELECT id, user_id, type, title, body, data, read, created_at
		FROM notifications WHERE user_id = ?
	`
	if unreadOnly {
		query += " AND read = 0"
	}
	query += " ORDER BY created_at DESC LIMIT 50"

	rows, err := s.db.QueryContext(ctx, query, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []store.Notification
	for rows.Next() {
		var n store.Notification
		var id, uID string
		var dataJSON sql.NullString
		var read int

		if err := rows.Scan(&id, &uID, &n.Type, &n.Title, &n.Body, &dataJSON, &read, &n.CreatedAt); err != nil {
			return nil, err
		}

		n.ID, _ = uuid.Parse(id)
		n.UserID, _ = uuid.Parse(uID)
		n.Read = read == 1

		if dataJSON.Valid {
			_ = json.Unmarshal([]byte(dataJSON.String), &n.Data)
		}

		notifications = append(notifications, n)
	}

	return notifications, rows.Err()
}

// MarkNotificationRead marks a notification as read
func (s *SocialStore) MarkNotificationRead(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET read = 1 WHERE id = ?
	`, id.String())
	return err
}
