package se

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/qa/store/duckdb"
)

func TestImportDir(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	writeFixture(t, filepath.Join(tmpDir, "Users.xml"), `<?xml version="1.0" encoding="utf-8"?>
<users>
  <row Id="1" DisplayName="Alice" Reputation="101" CreationDate="2024-01-01T00:00:00.000" LastAccessDate="2024-01-02T00:00:00.000" />
  <row Id="2" DisplayName="Bob" Reputation="50" CreationDate="2024-01-01T00:00:00.000" />
</users>
`)
	writeFixture(t, filepath.Join(tmpDir, "Tags.xml"), `<?xml version="1.0" encoding="utf-8"?>
<tags>
  <row Id="1" TagName="ai" Count="1" ExcerptPostId="10" WikiPostId="11" CreationDate="2024-01-01T00:00:00.000" />
</tags>
`)
	writeFixture(t, filepath.Join(tmpDir, "Posts.xml"), `<?xml version="1.0" encoding="utf-8"?>
<posts>
  <row Id="100" PostTypeId="1" OwnerUserId="1" Title="Question title" Body="&lt;p&gt;Question body&lt;/p&gt;" Tags="&lt;ai&gt;&lt;ml&gt;" CreationDate="2024-01-03T00:00:00.000" Score="5" ViewCount="10" AnswerCount="1" CommentCount="0" FavoriteCount="1" AcceptedAnswerId="101" />
  <row Id="101" PostTypeId="2" ParentId="100" OwnerUserId="2" Body="&lt;p&gt;Answer body&lt;/p&gt;" CreationDate="2024-01-03T00:00:00.000" Score="3" />
  <row Id="10" PostTypeId="5" Body="Tag excerpt body" CreationDate="2024-01-03T00:00:00.000" />
  <row Id="11" PostTypeId="4" Body="Tag wiki body" CreationDate="2024-01-03T00:00:00.000" />
</posts>
`)
	writeFixture(t, filepath.Join(tmpDir, "Comments.xml"), `<?xml version="1.0" encoding="utf-8"?>
<comments>
  <row Id="200" PostId="100" UserId="2" Text="Nice" Score="1" CreationDate="2024-01-04T00:00:00.000" />
</comments>
`)
	writeFixture(t, filepath.Join(tmpDir, "Votes.xml"), `<?xml version="1.0" encoding="utf-8"?>
<votes>
  <row Id="300" PostId="100" UserId="2" VoteTypeId="2" CreationDate="2024-01-04T00:00:00.000" />
  <row Id="301" PostId="100" UserId="1" VoteTypeId="5" CreationDate="2024-01-04T00:00:00.000" />
</votes>
`)
	writeFixture(t, filepath.Join(tmpDir, "Badges.xml"), `<?xml version="1.0" encoding="utf-8"?>
<badges>
  <row Id="400" UserId="1" Name="Teacher" Date="2024-01-05T00:00:00.000" Class="2" />
</badges>
`)

	store, err := duckdb.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	summary, err := NewImporter(store.DB()).ImportDir(ctx, tmpDir)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if summary.Users != 2 {
		t.Fatalf("expected 2 users, got %d", summary.Users)
	}
	if summary.Questions != 1 || summary.Answers != 1 {
		t.Fatalf("expected 1 question/answer, got %d/%d", summary.Questions, summary.Answers)
	}
	if summary.Comments != 1 {
		t.Fatalf("expected 1 comment, got %d", summary.Comments)
	}
	if summary.Votes != 1 || summary.Bookmarks != 1 {
		t.Fatalf("expected 1 vote/bookmark, got %d/%d", summary.Votes, summary.Bookmarks)
	}

	assertQuery(t, store.DB(), "SELECT COUNT(*) FROM tags WHERE name = 'ai'", 1)
	assertQuery(t, store.DB(), "SELECT COUNT(*) FROM tags WHERE name = 'ml'", 1)
	assertQuery(t, store.DB(), "SELECT COUNT(*) FROM question_tags", 2)
	assertQuery(t, store.DB(), "SELECT COUNT(*) FROM comments", 1)
	assertQuery(t, store.DB(), "SELECT COUNT(*) FROM votes", 1)
	assertQuery(t, store.DB(), "SELECT COUNT(*) FROM bookmarks", 1)

	var accepted string
	if err := store.DB().QueryRow("SELECT accepted_answer_id FROM questions WHERE id = 'se-question-100'").Scan(&accepted); err != nil {
		t.Fatalf("accepted answer: %v", err)
	}
	if accepted != "se-answer-101" {
		t.Fatalf("expected accepted answer se-answer-101, got %s", accepted)
	}

	var isAccepted bool
	if err := store.DB().QueryRow("SELECT is_accepted FROM answers WHERE id = 'se-answer-101'").Scan(&isAccepted); err != nil {
		t.Fatalf("answer accepted: %v", err)
	}
	if !isAccepted {
		t.Fatalf("expected answer accepted")
	}

	var excerpt, wiki string
	if err := store.DB().QueryRow("SELECT excerpt, wiki FROM tags WHERE name = 'ai'").Scan(&excerpt, &wiki); err != nil {
		t.Fatalf("tag content: %v", err)
	}
	if excerpt != "Tag excerpt body" || wiki != "Tag wiki body" {
		t.Fatalf("unexpected tag content: excerpt=%q wiki=%q", excerpt, wiki)
	}
}

func writeFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func assertQuery(t *testing.T, db *sql.DB, query string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if got != want {
		t.Fatalf("query %q: expected %d, got %d", query, want, got)
	}
}
