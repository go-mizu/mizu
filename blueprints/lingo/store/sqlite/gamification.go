package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// GamificationStore handles gamification operations
type GamificationStore struct {
	db *sql.DB
}

// GetLeagues returns all leagues
func (s *GamificationStore) GetLeagues(ctx context.Context) ([]store.League, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, icon_url, min_xp_to_promote, demotion_zone_size
		FROM leagues ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leagues []store.League
	for rows.Next() {
		var league store.League
		var iconURL sql.NullString

		if err := rows.Scan(&league.ID, &league.Name, &iconURL, &league.MinXPToPromote, &league.DemotionZoneSize); err != nil {
			return nil, err
		}

		if iconURL.Valid {
			league.IconURL = iconURL.String
		}

		leagues = append(leagues, league)
	}

	return leagues, rows.Err()
}

// GetCurrentSeason returns or creates the current week's season
func (s *GamificationStore) GetCurrentSeason(ctx context.Context, leagueID int) (*store.LeagueSeason, error) {
	// Calculate current week boundaries
	now := time.Now()
	weekStart := now.Truncate(24*time.Hour).AddDate(0, 0, -int(now.Weekday()))
	weekEnd := weekStart.AddDate(0, 0, 7).Add(-time.Second)

	// Try to get existing season
	var season store.LeagueSeason
	var id string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, league_id, week_start, week_end FROM league_seasons
		WHERE league_id = ? AND week_start = ?
	`, leagueID, weekStart).Scan(&id, &season.LeagueID, &season.WeekStart, &season.WeekEnd)

	if err == sql.ErrNoRows {
		// Create new season
		season.ID = uuid.New()
		season.LeagueID = leagueID
		season.WeekStart = weekStart
		season.WeekEnd = weekEnd

		_, err = s.db.ExecContext(ctx, `
			INSERT INTO league_seasons (id, league_id, week_start, week_end)
			VALUES (?, ?, ?, ?)
		`, season.ID.String(), leagueID, weekStart, weekEnd)
		if err != nil {
			return nil, err
		}

		return &season, nil
	}

	if err != nil {
		return nil, err
	}

	season.ID, _ = uuid.Parse(id)
	return &season, nil
}

// GetLeaderboard returns the leaderboard for a season
func (s *GamificationStore) GetLeaderboard(ctx context.Context, seasonID uuid.UUID, limit int) ([]store.UserLeague, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ul.id, ul.user_id, ul.season_id, ul.xp_earned, ul.rank, ul.promoted, ul.demoted,
			u.username, u.display_name, u.avatar_url, u.xp_total, u.streak_days
		FROM user_leagues ul
		JOIN users u ON ul.user_id = u.id
		WHERE ul.season_id = ?
		ORDER BY ul.xp_earned DESC
		LIMIT ?
	`, seasonID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaderboard []store.UserLeague
	rank := 0
	for rows.Next() {
		rank++
		var ul store.UserLeague
		var id, userID, sID string
		var promoted, demoted sql.NullInt64
		var user store.User
		var uID string

		if err := rows.Scan(&id, &userID, &sID, &ul.XPEarned, &ul.Rank, &promoted, &demoted,
			&user.Username, &user.DisplayName, &user.AvatarURL, &user.XPTotal, &user.StreakDays); err != nil {
			return nil, err
		}

		ul.ID, _ = uuid.Parse(id)
		ul.UserID, _ = uuid.Parse(userID)
		ul.SeasonID, _ = uuid.Parse(sID)
		ul.Rank = rank
		ul.Promoted = promoted.Valid && promoted.Int64 == 1
		ul.Demoted = demoted.Valid && demoted.Int64 == 1

		user.ID, _ = uuid.Parse(uID)
		ul.User = &user

		leaderboard = append(leaderboard, ul)
	}

	return leaderboard, rows.Err()
}

// GetUserLeague returns a user's current league participation
func (s *GamificationStore) GetUserLeague(ctx context.Context, userID uuid.UUID) (*store.UserLeague, error) {
	// Get the current week's start
	now := time.Now()
	weekStart := now.Truncate(24*time.Hour).AddDate(0, 0, -int(now.Weekday()))

	var ul store.UserLeague
	var id, uID, sID string
	var promoted, demoted sql.NullInt64

	err := s.db.QueryRowContext(ctx, `
		SELECT ul.id, ul.user_id, ul.season_id, ul.xp_earned, ul.rank, ul.promoted, ul.demoted
		FROM user_leagues ul
		JOIN league_seasons ls ON ul.season_id = ls.id
		WHERE ul.user_id = ? AND ls.week_start = ?
	`, userID.String(), weekStart).Scan(&id, &uID, &sID, &ul.XPEarned, &ul.Rank, &promoted, &demoted)

	if err != nil {
		return nil, err
	}

	ul.ID, _ = uuid.Parse(id)
	ul.UserID, _ = uuid.Parse(uID)
	ul.SeasonID, _ = uuid.Parse(sID)
	ul.Promoted = promoted.Valid && promoted.Int64 == 1
	ul.Demoted = demoted.Valid && demoted.Int64 == 1

	return &ul, nil
}

// JoinLeague joins a user to a league season
func (s *GamificationStore) JoinLeague(ctx context.Context, userID uuid.UUID, seasonID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_leagues (id, user_id, season_id, xp_earned, rank)
		VALUES (?, ?, ?, 0, 0)
	`, uuid.New().String(), userID.String(), seasonID.String())
	return err
}

// UpdateLeagueXP updates a user's league XP
func (s *GamificationStore) UpdateLeagueXP(ctx context.Context, userID, seasonID uuid.UUID, xp int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE user_leagues SET xp_earned = xp_earned + ? WHERE user_id = ? AND season_id = ?
	`, xp, userID.String(), seasonID.String())
	return err
}

// ProcessWeeklyLeagues processes weekly league transitions
func (s *GamificationStore) ProcessWeeklyLeagues(ctx context.Context) error {
	// This would be called by a scheduled job
	// For each season that just ended:
	// 1. Mark top 10 as promoted
	// 2. Mark bottom 5 as demoted
	// 3. Create new seasons for next week

	// For now, this is a placeholder
	return nil
}
