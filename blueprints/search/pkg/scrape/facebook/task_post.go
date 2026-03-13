package facebook

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type PostState struct {
	URL           string
	Status        string
	CommentsFound int
	Error         string
}

type PostMetric struct {
	Fetched  int
	Skipped  int
	Failed   int
	Comments int
}

type PostTask struct {
	URL         string
	Client      *Client
	DB          *DB
	StateDB     *State
	MaxComments int
}

var _ core.Task[PostState, PostMetric] = (*PostTask)(nil)

func (t *PostTask) Run(ctx context.Context, emit func(*PostState)) (PostMetric, error) {
	var m PostMetric
	target := NormalizePostURL(t.URL)
	emit(&PostState{URL: target, Status: "fetching"})

	doc, code, err := t.Client.FetchHTML(ctx, target)
	if err != nil {
		m.Failed++
		emit(&PostState{URL: target, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(target, err.Error())
		}
		return m, nil
	}
	if code == 404 || doc == nil {
		m.Skipped++
		emit(&PostState{URL: target, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(target, EntityPost, code)
		}
		return m, nil
	}

	postID := extractPostID(target)
	posts := ParsePosts(doc, target, "", "", EntityPost)
	if len(posts) == 0 && postID != "" {
		posts = []Post{{
			PostID:    postID,
			OwnerType: EntityPost,
			Text:      cleanText(doc.Find("body").Text()),
			Permalink: CanonicalURL(target),
			FetchedAt: now(),
		}}
	}
	for _, post := range posts {
		if post.PostID == "" {
			continue
		}
		_ = t.DB.UpsertPost(post)
		comments := ParseComments(doc, post.PostID, target, t.MaxComments)
		_ = t.DB.InsertComments(comments)
		m.Comments += len(comments)
	}
	if t.StateDB != nil {
		enqueueDiscoveredLinks(t.StateDB, doc, target)
		t.StateDB.Done(target, EntityPost, code)
	}

	m.Fetched++
	emit(&PostState{URL: target, Status: "done", CommentsFound: m.Comments})
	return m, nil
}
