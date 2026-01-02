package se

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultBatchSize = 5000
)

// Summary captures import totals.
type Summary struct {
	Users        int
	Tags         int
	Questions    int
	Answers      int
	Comments     int
	Votes        int
	Bookmarks    int
	Badges       int
	BadgeAwards  int
	QuestionTags int
	TagWikis     int
	TagExcerpts  int
}

// ProgressReporter receives progress updates during import.
type ProgressReporter interface {
	StartFile(name string, size int64)
	Advance(name string, rows int, bytesRead int64)
	EndFile(name string, rows int, duration time.Duration)
	Logf(format string, args ...any)
}

// NopProgress is a no-op progress reporter.
type NopProgress struct{}

func (NopProgress) StartFile(string, int64)            {}
func (NopProgress) Advance(string, int, int64)         {}
func (NopProgress) EndFile(string, int, time.Duration) {}
func (NopProgress) Logf(string, ...any)                {}

// Importer imports StackExchange dumps into DuckDB.
type Importer struct {
	db        *sql.DB
	batchSize int
	progress  ProgressReporter
}

// NewImporter creates a new importer.
func NewImporter(db *sql.DB) *Importer {
	return &Importer{
		db:        db,
		batchSize: defaultBatchSize,
		progress:  NopProgress{},
	}
}

// WithBatchSize sets the batch size for inserts.
func (i *Importer) WithBatchSize(n int) *Importer {
	if n > 0 {
		i.batchSize = n
	}
	return i
}

// WithProgress sets the progress reporter.
func (i *Importer) WithProgress(p ProgressReporter) *Importer {
	if p != nil {
		i.progress = p
	}
	return i
}

// ImportDir imports data from the given directory.
func (i *Importer) ImportDir(ctx context.Context, dir string) (Summary, error) {
	summary := Summary{}

	usersPath := filepath.Join(dir, "Users.xml")
	tagsPath := filepath.Join(dir, "Tags.xml")
	postsPath := filepath.Join(dir, "Posts.xml")
	commentsPath := filepath.Join(dir, "Comments.xml")
	votesPath := filepath.Join(dir, "Votes.xml")
	badgesPath := filepath.Join(dir, "Badges.xml")

	tagWikiByPost := make(map[int64]string)
	tagExcerptByPost := make(map[int64]string)
	postTypes := make(map[int64]postType)
	acceptedAnswerByQuestion := make(map[int64]int64)
	tagCounts := make(map[string]int64)

	if err := i.importUsers(ctx, usersPath, &summary); err != nil {
		return summary, err
	}
	if err := i.importTags(ctx, tagsPath, &summary, tagWikiByPost, tagExcerptByPost); err != nil {
		return summary, err
	}
	if err := i.importPosts(ctx, postsPath, &summary, postTypes, acceptedAnswerByQuestion, tagCounts, tagWikiByPost, tagExcerptByPost); err != nil {
		return summary, err
	}
	if err := i.importComments(ctx, commentsPath, &summary, postTypes); err != nil {
		return summary, err
	}
	if err := i.importVotes(ctx, votesPath, &summary, postTypes); err != nil {
		return summary, err
	}
	if err := i.importBadges(ctx, badgesPath, &summary); err != nil {
		return summary, err
	}
	if err := i.updateAcceptedAnswers(ctx, acceptedAnswerByQuestion); err != nil {
		return summary, err
	}

	return summary, nil
}

func (i *Importer) importUsers(ctx context.Context, path string, summary *Summary) error {
	return i.importRows(ctx, path, "Users.xml", func(tx *sql.Tx, batch *batchState) (func(), error) {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO accounts (
				id, username, email, password_hash, display_name, bio,
				avatar_url, location, website_url, reputation,
				is_moderator, is_admin, is_suspended, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return nil, err
		}
		cleanup := func() { _ = stmt.Close() }

		if _, err := stmt.ExecContext(ctx,
			userID(0),
			userUsername(0, "Community"),
			userEmail(0),
			defaultPasswordHash,
			"Community",
			"",
			"",
			"",
			"",
			int64(1),
			false,
			false,
			false,
			time.Now().UTC(),
			time.Now().UTC(),
		); err != nil {
			cleanup()
			return nil, err
		}

		batch.handleUser = func(row userRow) error {
			if row.ID == 0 {
				return nil
			}
			createdAt := parseTime(row.CreationDate)
			updatedAt := parseTime(row.LastAccessDate)
			if updatedAt.IsZero() {
				updatedAt = createdAt
			}
			_, err := stmt.ExecContext(ctx,
				userID(row.ID),
				userUsername(row.ID, row.DisplayName),
				userEmail(row.ID),
				defaultPasswordHash,
				row.DisplayName,
				stripHTML(row.AboutMe),
				rawOrEmpty(row.ProfileImageURL),
				rawOrEmpty(row.Location),
				rawOrEmpty(row.WebsiteURL),
				row.Reputation,
				false,
				false,
				false,
				createdAt,
				updatedAt,
			)
			if err != nil {
				return err
			}
			summary.Users++
			return nil
		}
		return cleanup, nil
	})
}

func (i *Importer) importTags(ctx context.Context, path string, summary *Summary, tagWikiByPost, tagExcerptByPost map[int64]string) error {
	return i.importRows(ctx, path, "Tags.xml", func(tx *sql.Tx, batch *batchState) (func(), error) {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO tags (id, name, excerpt, wiki, question_count, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (name) DO NOTHING
		`)
		if err != nil {
			return nil, err
		}
		cleanup := func() { _ = stmt.Close() }

		batch.handleTag = func(row tagRow) error {
			if row.ID == 0 || row.TagName == "" {
				return nil
			}
			name := strings.ToLower(strings.TrimSpace(row.TagName))
			if name == "" {
				return nil
			}
			createdAt := parseTime(row.CreationDate)
			_, err := stmt.ExecContext(ctx,
				tagID(name),
				name,
				rawOrEmpty(row.Excerpt),
				rawOrEmpty(row.Wiki),
				row.Count,
				createdAt,
			)
			if err != nil {
				return err
			}
			if row.ExcerptPostID != 0 {
				tagExcerptByPost[row.ExcerptPostID] = name
			}
			if row.WikiPostID != 0 {
				tagWikiByPost[row.WikiPostID] = name
			}
			summary.Tags++
			return nil
		}
		return cleanup, nil
	})
}

func (i *Importer) importPosts(
	ctx context.Context,
	path string,
	summary *Summary,
	postTypes map[int64]postType,
	acceptedAnswerByQuestion map[int64]int64,
	tagCounts map[string]int64,
	tagWikiByPost map[int64]string,
	tagExcerptByPost map[int64]string,
) error {
	tagWikiByName := make(map[string]string)
	tagExcerptByName := make(map[string]string)

	if err := i.importRows(ctx, path, "Posts.xml", func(tx *sql.Tx, batch *batchState) (func(), error) {
		questionStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO questions (
				id, author_id, title, body, body_html, score, view_count,
				answer_count, comment_count, favorite_count, accepted_answer_id,
				bounty_amount, is_closed, close_reason, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return nil, err
		}

		tagStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO tags (id, name, excerpt, wiki, question_count, created_at)
			VALUES ($1, $2, '', '', 0, $3)
			ON CONFLICT (name) DO NOTHING
		`)
		if err != nil {
			_ = questionStmt.Close()
			return nil, err
		}
		cleanup := func() {
			_ = questionStmt.Close()
			_ = tagStmt.Close()
		}

		batch.handlePost = func(row postRow) error {
			if row.ID == 0 {
				return nil
			}
			createdAt := parseTime(row.CreationDate)
			updatedAt := parseTime(row.LastEditDate)
			if updatedAt.IsZero() {
				updatedAt = createdAt
			}

			switch postType(row.PostTypeID) {
			case postTypeQuestion:
				postTypes[row.ID] = postTypeQuestion
				if row.AcceptedAnswerID != 0 {
					acceptedAnswerByQuestion[row.ID] = row.AcceptedAnswerID
				}
				bodyText := stripHTML(row.Body)
				isClosed := strings.TrimSpace(row.ClosedReason) != ""
				_, err := questionStmt.ExecContext(ctx,
					questionID(row.ID),
					userID(row.OwnerUserID),
					rawOrEmpty(row.Title),
					bodyText,
					rawOrEmpty(row.Body),
					row.Score,
					row.ViewCount,
					row.AnswerCount,
					row.CommentCount,
					row.FavoriteCount,
					"",
					int64(0),
					isClosed,
					rawOrEmpty(row.ClosedReason),
					createdAt,
					updatedAt,
				)
				if err != nil {
					return err
				}

				tags := parseTags(row.Tags)
				for _, tag := range tags {
					if tag == "" {
						continue
					}
					if _, err := tagStmt.ExecContext(ctx, tagID(tag), tag, createdAt); err != nil {
						return err
					}
					tagCounts[tag]++
				}

				summary.Questions++
			case postTypeTagWiki:
				if tagName, ok := tagWikiByPost[row.ID]; ok {
					tagWikiByName[tagName] = rawOrEmpty(row.Body)
					summary.TagWikis++
				}
			case postTypeTagExcerpt:
				if tagName, ok := tagExcerptByPost[row.ID]; ok {
					tagExcerptByName[tagName] = rawOrEmpty(row.Body)
					summary.TagExcerpts++
				}
			}
			return nil
		}
		return cleanup, nil
	}); err != nil {
		return err
	}

	if err := i.updateTagContent(ctx, tagExcerptByName, tagWikiByName); err != nil {
		return err
	}
	if err := i.updateTagCounts(ctx, tagCounts); err != nil {
		return err
	}

	if err := i.importRows(ctx, path, "Posts.xml", func(tx *sql.Tx, batch *batchState) (func(), error) {
		answerStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO answers (
				id, question_id, author_id, body, body_html, score,
				is_accepted, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return nil, err
		}
		cleanup := func() { _ = answerStmt.Close() }

		batch.handlePost = func(row postRow) error {
			if row.ID == 0 {
				return nil
			}
			if postType(row.PostTypeID) != postTypeAnswer {
				return nil
			}
			postTypes[row.ID] = postTypeAnswer
			if row.ParentID == 0 {
				return nil
			}
			createdAt := parseTime(row.CreationDate)
			updatedAt := parseTime(row.LastEditDate)
			if updatedAt.IsZero() {
				updatedAt = createdAt
			}
			bodyText := stripHTML(row.Body)
			_, err := answerStmt.ExecContext(ctx,
				answerID(row.ID),
				questionID(row.ParentID),
				userID(row.OwnerUserID),
				bodyText,
				rawOrEmpty(row.Body),
				row.Score,
				false,
				createdAt,
				updatedAt,
			)
			if err != nil {
				return err
			}
			summary.Answers++
			return nil
		}
		return cleanup, nil
	}); err != nil {
		return err
	}

	return i.importRows(ctx, path, "Posts.xml", func(tx *sql.Tx, batch *batchState) (func(), error) {
		linkStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO question_tags (question_id, tag_id)
			VALUES ($1, $2)
			ON CONFLICT (question_id, tag_id) DO NOTHING
		`)
		if err != nil {
			return nil, err
		}
		cleanup := func() { _ = linkStmt.Close() }

		batch.handlePost = func(row postRow) error {
			if row.ID == 0 {
				return nil
			}
			if postType(row.PostTypeID) != postTypeQuestion {
				return nil
			}
			tags := parseTags(row.Tags)
			for _, tag := range tags {
				if tag == "" {
					continue
				}
				if _, err := linkStmt.ExecContext(ctx, questionID(row.ID), tagID(tag)); err != nil {
					return err
				}
				summary.QuestionTags++
			}
			return nil
		}
		return cleanup, nil
	})
}

func (i *Importer) importComments(ctx context.Context, path string, summary *Summary, postTypes map[int64]postType) error {
	return i.importRows(ctx, path, "Comments.xml", func(tx *sql.Tx, batch *batchState) (func(), error) {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO comments (
				id, target_type, target_id, author_id, body, score, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return nil, err
		}
		cleanup := func() { _ = stmt.Close() }

		batch.handleComment = func(row commentRow) error {
			if row.ID == 0 || row.PostID == 0 {
				return nil
			}
			pType, ok := postTypes[row.PostID]
			if !ok {
				return nil
			}
			targetType, targetID := targetForPost(pType, row.PostID)
			createdAt := parseTime(row.CreationDate)
			updatedAt := createdAt
			_, err := stmt.ExecContext(ctx,
				commentID(row.ID),
				targetType,
				targetID,
				userID(row.UserID),
				rawOrEmpty(row.Text),
				row.Score,
				createdAt,
				updatedAt,
			)
			if err != nil {
				return err
			}
			summary.Comments++
			return nil
		}
		return cleanup, nil
	})
}

func (i *Importer) importVotes(ctx context.Context, path string, summary *Summary, postTypes map[int64]postType) error {
	return i.importRows(ctx, path, "Votes.xml", func(tx *sql.Tx, batch *batchState) (func(), error) {
		voteStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO votes (
				id, voter_id, target_type, target_id, value, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (voter_id, target_type, target_id) DO NOTHING
		`)
		if err != nil {
			return nil, err
		}

		bookmarkStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO bookmarks (id, account_id, question_id, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (account_id, question_id) DO NOTHING
		`)
		if err != nil {
			_ = voteStmt.Close()
			return nil, err
		}
		cleanup := func() {
			_ = voteStmt.Close()
			_ = bookmarkStmt.Close()
		}

		batch.handleVote = func(row voteRow) error {
			if row.ID == 0 || row.PostID == 0 {
				return nil
			}
			createdAt := parseTime(row.CreationDate)
			voterID := userID(row.UserID)
			pType, ok := postTypes[row.PostID]
			if !ok {
				return nil
			}
			if row.VoteTypeID == voteTypeFavorite {
				if pType != postTypeQuestion {
					return nil
				}
				_, err := bookmarkStmt.ExecContext(ctx,
					bookmarkID(row.ID),
					voterID,
					questionID(row.PostID),
					createdAt,
				)
				if err != nil {
					return err
				}
				summary.Bookmarks++
				return nil
			}

			value, ok := voteValue(row.VoteTypeID)
			if !ok {
				return nil
			}
			targetType, targetID := targetForPost(pType, row.PostID)
			_, err := voteStmt.ExecContext(ctx,
				voteID(row.ID),
				voterID,
				targetType,
				targetID,
				value,
				createdAt,
				createdAt,
			)
			if err != nil {
				return err
			}
			summary.Votes++
			return nil
		}
		return cleanup, nil
	})
}

func (i *Importer) importBadges(ctx context.Context, path string, summary *Summary) error {
	return i.importRows(ctx, path, "Badges.xml", func(tx *sql.Tx, batch *batchState) (func(), error) {
		seenBadges := make(map[string]bool)
		badgeStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO badges (id, name, tier, description)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (name) DO NOTHING
		`)
		if err != nil {
			return nil, err
		}

		awardStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO badge_awards (id, account_id, badge_id, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			_ = badgeStmt.Close()
			return nil, err
		}
		cleanup := func() {
			_ = badgeStmt.Close()
			_ = awardStmt.Close()
		}

		batch.handleBadge = func(row badgeRow) error {
			if row.ID == 0 || row.UserID == 0 || row.Name == "" {
				return nil
			}
			badgeIDVal := badgeID(row.Name)
			tier := badgeTier(row.Class)
			_, err := badgeStmt.ExecContext(ctx, badgeIDVal, row.Name, tier, "")
			if err != nil {
				return err
			}
			if !seenBadges[row.Name] {
				seenBadges[row.Name] = true
				summary.Badges++
			}
			_, err = awardStmt.ExecContext(ctx,
				badgeAwardID(row.ID),
				userID(row.UserID),
				badgeIDVal,
				parseTime(row.Date),
			)
			if err != nil {
				return err
			}
			summary.BadgeAwards++
			return nil
		}
		return cleanup, nil
	})
}

func (i *Importer) updateAcceptedAnswers(ctx context.Context, acceptedAnswerByQuestion map[int64]int64) error {
	if len(acceptedAnswerByQuestion) == 0 {
		return nil
	}
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	questionStmt, err := tx.PrepareContext(ctx, `
		UPDATE questions SET accepted_answer_id = $2 WHERE id = $1
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer questionStmt.Close()

	answerStmt, err := tx.PrepareContext(ctx, `
		UPDATE answers SET is_accepted = TRUE WHERE id = $1
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer answerStmt.Close()

	for qID, aID := range acceptedAnswerByQuestion {
		questionIDVal := questionID(qID)
		answerIDVal := answerID(aID)
		if _, err = questionStmt.ExecContext(ctx, questionIDVal, answerIDVal); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err = answerStmt.ExecContext(ctx, answerIDVal); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (i *Importer) updateTagContent(ctx context.Context, excerpts, wikis map[string]string) error {
	if len(excerpts) == 0 && len(wikis) == 0 {
		return nil
	}
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	excerptStmt, err := tx.PrepareContext(ctx, `
		UPDATE tags SET excerpt = $2 WHERE name = $1
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer excerptStmt.Close()

	wikiStmt, err := tx.PrepareContext(ctx, `
		UPDATE tags SET wiki = $2 WHERE name = $1
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer wikiStmt.Close()

	for name, body := range excerpts {
		if _, err := excerptStmt.ExecContext(ctx, name, body); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	for name, body := range wikis {
		if _, err := wikiStmt.ExecContext(ctx, name, body); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (i *Importer) updateTagCounts(ctx context.Context, tagCounts map[string]int64) error {
	if len(tagCounts) == 0 {
		return nil
	}
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `
		UPDATE tags SET question_count = $2 WHERE name = $1
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for name, count := range tagCounts {
		if _, err := stmt.ExecContext(ctx, name, count); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

type batchState struct {
	handleUser    func(userRow) error
	handleTag     func(tagRow) error
	handlePost    func(postRow) error
	handleComment func(commentRow) error
	handleVote    func(voteRow) error
	handleBadge   func(badgeRow) error
}

func (i *Importer) importRows(ctx context.Context, path, name string, setup func(tx *sql.Tx, batch *batchState) (func(), error)) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			i.progress.Logf("skip %s (not found)", name)
			return nil
		}
		return err
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	i.progress.StartFile(name, info.Size())
	start := time.Now()

	reader := bufio.NewReaderSize(f, 1024*1024)
	counting := &countingReader{r: reader}
	decoder := xml.NewDecoder(counting)

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	batch := &batchState{}
	cleanup, err := setup(tx, batch)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	rows := 0
	flushBatch := func() error {
		if cleanup != nil {
			cleanup()
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		tx, err = i.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		batch = &batchState{}
		cleanup, err = setup(tx, batch)
		return err
	}

	lastReport := time.Now()
	for {
		tok, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			_ = tx.Rollback()
			return err
		}
		startElem, ok := tok.(xml.StartElement)
		if !ok || startElem.Name.Local != "row" {
			continue
		}

		switch name {
		case "Users.xml":
			var row userRow
			if err := decoder.DecodeElement(&row, &startElem); err != nil {
				_ = tx.Rollback()
				return err
			}
			if batch.handleUser != nil {
				if err := batch.handleUser(row); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		case "Tags.xml":
			var row tagRow
			if err := decoder.DecodeElement(&row, &startElem); err != nil {
				_ = tx.Rollback()
				return err
			}
			if batch.handleTag != nil {
				if err := batch.handleTag(row); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		case "Posts.xml":
			var row postRow
			if err := decoder.DecodeElement(&row, &startElem); err != nil {
				_ = tx.Rollback()
				return err
			}
			if batch.handlePost != nil {
				if err := batch.handlePost(row); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		case "Comments.xml":
			var row commentRow
			if err := decoder.DecodeElement(&row, &startElem); err != nil {
				_ = tx.Rollback()
				return err
			}
			if batch.handleComment != nil {
				if err := batch.handleComment(row); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		case "Votes.xml":
			var row voteRow
			if err := decoder.DecodeElement(&row, &startElem); err != nil {
				_ = tx.Rollback()
				return err
			}
			if batch.handleVote != nil {
				if err := batch.handleVote(row); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		case "Badges.xml":
			var row badgeRow
			if err := decoder.DecodeElement(&row, &startElem); err != nil {
				_ = tx.Rollback()
				return err
			}
			if batch.handleBadge != nil {
				if err := batch.handleBadge(row); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		}

		rows++
		if rows%i.batchSize == 0 {
			if err := flushBatch(); err != nil {
				return err
			}
		}

		if time.Since(lastReport) > 200*time.Millisecond {
			i.progress.Advance(name, rows, counting.bytes)
			lastReport = time.Now()
		}
	}

	if cleanup != nil {
		cleanup()
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	i.progress.Advance(name, rows, counting.bytes)
	i.progress.EndFile(name, rows, time.Since(start))
	return nil
}

func targetForPost(pType postType, postID int64) (string, string) {
	if pType == postTypeQuestion {
		return "question", questionID(postID)
	}
	return "answer", answerID(postID)
}

type countingReader struct {
	r     io.Reader
	bytes int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.bytes += int64(n)
	return n, err
}

func rawOrEmpty(val string) string {
	return val
}

// ImportQuestionTags imports only the question-tag associations from Posts.xml.
// This is useful for fixing a database where question_tags is empty.
func (i *Importer) ImportQuestionTags(ctx context.Context, postsPath string) (int, error) {
	count := 0

	err := i.importRows(ctx, postsPath, "Posts.xml", func(tx *sql.Tx, batch *batchState) (func(), error) {
		linkStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO question_tags (question_id, tag_id)
			VALUES ($1, $2)
			ON CONFLICT (question_id, tag_id) DO NOTHING
		`)
		if err != nil {
			return nil, err
		}
		cleanup := func() { _ = linkStmt.Close() }

		batch.handlePost = func(row postRow) error {
			if row.ID == 0 {
				return nil
			}
			if postType(row.PostTypeID) != postTypeQuestion {
				return nil
			}
			tags := parseTags(row.Tags)
			for _, tag := range tags {
				if tag == "" {
					continue
				}
				if _, err := linkStmt.ExecContext(ctx, questionID(row.ID), tagID(tag)); err != nil {
					return err
				}
				count++
			}
			return nil
		}
		return cleanup, nil
	})

	return count, err
}
