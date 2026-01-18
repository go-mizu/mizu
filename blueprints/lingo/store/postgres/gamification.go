package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GamificationStore implements store.GamificationStore
type GamificationStore struct {
	pool *pgxpool.Pool
}

// GetLeagues gets all leagues
func (s *GamificationStore) GetLeagues(ctx context.Context) ([]store.League, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, icon_url, min_xp_to_promote, demotion_zone_size
		FROM leagues ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("query leagues: %w", err)
	}
	defer rows.Close()

	var leagues []store.League
	for rows.Next() {
		var l store.League
		if err := rows.Scan(&l.ID, &l.Name, &l.IconURL, &l.MinXPToPromote, &l.DemotionZoneSize); err != nil {
			return nil, fmt.Errorf("scan league: %w", err)
		}
		leagues = append(leagues, l)
	}
	return leagues, nil
}

// GetCurrentSeason gets the current season for a league
func (s *GamificationStore) GetCurrentSeason(ctx context.Context, leagueID int) (*store.LeagueSeason, error) {
	season := &store.LeagueSeason{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, league_id, week_start, week_end
		FROM league_seasons WHERE league_id = $1 AND week_start <= CURRENT_DATE AND week_end >= CURRENT_DATE
	`, leagueID).Scan(&season.ID, &season.LeagueID, &season.WeekStart, &season.WeekEnd)
	if err != nil {
		// Create new season if none exists
		now := time.Now()
		weekStart := now.AddDate(0, 0, -int(now.Weekday()))
		weekEnd := weekStart.AddDate(0, 0, 6)
		season.ID = uuid.New()
		season.LeagueID = leagueID
		season.WeekStart = weekStart
		season.WeekEnd = weekEnd

		_, err = s.pool.Exec(ctx, `
			INSERT INTO league_seasons (id, league_id, week_start, week_end)
			VALUES ($1, $2, $3, $4)
		`, season.ID, season.LeagueID, season.WeekStart, season.WeekEnd)
		if err != nil {
			return nil, fmt.Errorf("create season: %w", err)
		}
	}
	return season, nil
}

// GetLeaderboard gets the leaderboard for a season
func (s *GamificationStore) GetLeaderboard(ctx context.Context, seasonID uuid.UUID, limit int) ([]store.UserLeague, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ul.id, ul.user_id, ul.season_id, ul.xp_earned, ul.rank, ul.promoted, ul.demoted,
		       u.id, u.username, u.display_name, u.avatar_url, u.xp_total
		FROM user_leagues ul
		JOIN users u ON ul.user_id = u.id
		WHERE ul.season_id = $1
		ORDER BY ul.xp_earned DESC
		LIMIT $2
	`, seasonID, limit)
	if err != nil {
		return nil, fmt.Errorf("query leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []store.UserLeague
	rank := 1
	for rows.Next() {
		var ul store.UserLeague
		var u store.User
		if err := rows.Scan(&ul.ID, &ul.UserID, &ul.SeasonID, &ul.XPEarned, &ul.Rank, &ul.Promoted, &ul.Demoted,
			&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.XPTotal); err != nil {
			return nil, fmt.Errorf("scan user league: %w", err)
		}
		ul.Rank = rank
		ul.User = &u
		entries = append(entries, ul)
		rank++
	}
	return entries, nil
}

// GetUserLeague gets a user's current league participation
func (s *GamificationStore) GetUserLeague(ctx context.Context, userID uuid.UUID) (*store.UserLeague, error) {
	ul := &store.UserLeague{}
	err := s.pool.QueryRow(ctx, `
		SELECT ul.id, ul.user_id, ul.season_id, ul.xp_earned, ul.rank, ul.promoted, ul.demoted
		FROM user_leagues ul
		JOIN league_seasons ls ON ul.season_id = ls.id
		WHERE ul.user_id = $1 AND ls.week_start <= CURRENT_DATE AND ls.week_end >= CURRENT_DATE
		ORDER BY ls.week_start DESC
		LIMIT 1
	`, userID).Scan(&ul.ID, &ul.UserID, &ul.SeasonID, &ul.XPEarned, &ul.Rank, &ul.Promoted, &ul.Demoted)
	if err != nil {
		return nil, fmt.Errorf("query user league: %w", err)
	}
	return ul, nil
}

// JoinLeague adds a user to a league season
func (s *GamificationStore) JoinLeague(ctx context.Context, userID uuid.UUID, seasonID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_leagues (id, user_id, season_id, xp_earned, rank)
		VALUES ($1, $2, $3, 0, 0)
		ON CONFLICT (user_id, season_id) DO NOTHING
	`, uuid.New(), userID, seasonID)
	if err != nil {
		return fmt.Errorf("join league: %w", err)
	}
	return nil
}

// UpdateLeagueXP updates a user's XP in their league
func (s *GamificationStore) UpdateLeagueXP(ctx context.Context, userID, seasonID uuid.UUID, xp int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_leagues SET xp_earned = xp_earned + $3
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID, xp)
	if err != nil {
		return fmt.Errorf("update league xp: %w", err)
	}
	return nil
}

// ProcessWeeklyLeagues processes weekly league promotions/demotions
func (s *GamificationStore) ProcessWeeklyLeagues(ctx context.Context) error {
	// This would be called by a cron job at the end of each week
	// to promote top performers and demote bottom performers
	return nil
}
