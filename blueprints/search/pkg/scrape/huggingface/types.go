package huggingface

import "time"

type RepoFile struct {
	EntityType string
	RepoID     string
	Path       string
	Size       int64
	LFSJSON    string
}

type RepoLink struct {
	SrcType string
	SrcID   string
	Rel     string
	DstType string
	DstID   string
}

type CollectionItem struct {
	CollectionSlug string
	ItemID         string
	ItemType       string
	Position       int
	Author         string
	RepoType       string
	RawJSON        string
}

type Model struct {
	RepoID               string
	Author               string
	SHA                  string
	CreatedAt            time.Time
	LastModified         time.Time
	Private              bool
	Gated                bool
	Disabled             bool
	Likes                int64
	Downloads            int64
	TrendingScore        int64
	PipelineTag          string
	LibraryName          string
	TagsJSON             string
	CardDataJSON         string
	ConfigJSON           string
	TransformersInfoJSON string
	WidgetDataJSON       string
	SpacesJSON           string
	RawJSON              string
	FetchedAt            time.Time
}

type Dataset struct {
	RepoID        string
	Author        string
	SHA           string
	CreatedAt     time.Time
	LastModified  time.Time
	Private       bool
	Gated         bool
	Disabled      bool
	Likes         int64
	Downloads     int64
	TrendingScore int64
	Description   string
	TagsJSON      string
	CardDataJSON  string
	RawJSON       string
	FetchedAt     time.Time
}

type Space struct {
	RepoID       string
	Author       string
	SHA          string
	CreatedAt    time.Time
	LastModified time.Time
	Private      bool
	Disabled     bool
	Likes        int64
	SDK          string
	Subdomain    string
	TagsJSON     string
	RuntimeJSON  string
	CardDataJSON string
	RawJSON      string
	FetchedAt    time.Time
}

type Collection struct {
	Slug        string
	Namespace   string
	Title       string
	Description string
	OwnerJSON   string
	Theme       string
	Upvotes     int64
	Private     bool
	Gating      bool
	LastUpdated time.Time
	ItemsJSON   string
	RawJSON     string
	FetchedAt   time.Time
}

type Paper struct {
	PaperID      string
	Title        string
	Summary      string
	AISummary    string
	PublishedAt  time.Time
	Upvotes      int64
	AuthorsJSON  string
	GitHubRepo   string
	ProjectPage  string
	ThumbnailURL string
	RawJSON      string
	FetchedAt    time.Time
}

type QueueItem struct {
	ID         int64
	URL        string
	EntityType string
	Priority   int
}

type JobRecord struct {
	JobID       string
	Name        string
	Type        string
	Status      string
	StartedAt   time.Time
	CompletedAt time.Time
}

type DBStats struct {
	Models      int64
	Datasets    int64
	Spaces      int64
	Collections int64
	Papers      int64
	RepoFiles   int64
	RepoLinks   int64
	DBSize      int64
}

type ResultMetric struct {
	Fetched int
	Skipped int
	Failed  int
}
