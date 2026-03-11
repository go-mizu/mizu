package apify

import "time"

// StoreSearchResponse is Algolia's search response for prod_PUBLIC_STORE.
type StoreSearchResponse struct {
	Hits         []StoreActorHit `json:"hits"`
	NbHits       int             `json:"nbHits"`
	Page         int             `json:"page"`
	NbPages      int             `json:"nbPages"`
	HitsPerPage  int             `json:"hitsPerPage"`
	Exhaustive   map[string]any  `json:"exhaustive,omitempty"`
	Query        string          `json:"query,omitempty"`
	Params       string          `json:"params,omitempty"`
	ProcessingMS int             `json:"processingTimeMS,omitempty"`
}

// StoreActorHit is the actor row from Algolia store index.
type StoreActorHit struct {
	ObjectID    string   `json:"objectID"`
	Name        string   `json:"name"`
	Username    string   `json:"username"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Categories  []string `json:"categories"`
	ModifiedAt  any      `json:"modifiedAt,omitempty"`
	CreatedAt   any      `json:"createdAt,omitempty"`
	PictureURL  string   `json:"pictureUrl,omitempty"`
}

// ActorDetailResponse is /v2/acts/{id} response.
type ActorDetailResponse struct {
	Data *ActorDetail `json:"data"`
}

// ActorDetail mirrors key actor fields while preserving room for raw JSON storage.
type ActorDetail struct {
	ID                 string           `json:"id"`
	UserID             string           `json:"userId"`
	Name               string           `json:"name"`
	Username           string           `json:"username"`
	Title              string           `json:"title"`
	Description        string           `json:"description"`
	Notice             string           `json:"notice"`
	ReadmeSummary      string           `json:"readmeSummary"`
	ActorPermission    string           `json:"actorPermissionLevel"`
	DeploymentKey      string           `json:"deploymentKey"`
	StandbyURL         string           `json:"standbyUrl"`
	PictureURL         string           `json:"pictureUrl"`
	SEODescription     string           `json:"seoDescription"`
	SEOTitle           string           `json:"seoTitle"`
	IsPublic           bool             `json:"isPublic"`
	IsDeprecated       bool             `json:"isDeprecated"`
	IsGeneric          bool             `json:"isGeneric"`
	IsCritical         bool             `json:"isCritical"`
	IsSourceCodeHidden bool             `json:"isSourceCodeHidden"`
	HasNoDataset       bool             `json:"hasNoDataset"`
	CreatedAt          time.Time        `json:"createdAt"`
	ModifiedAt         time.Time        `json:"modifiedAt"`
	Categories         []string         `json:"categories"`
	Stats              map[string]any   `json:"stats"`
	PricingInfos       []map[string]any `json:"pricingInfos"`
	Versions           []map[string]any `json:"versions"`
	VersionsAll        []map[string]any `json:"versionsAll,omitempty"`
	DefaultRunOptions  map[string]any   `json:"defaultRunOptions"`
	ExampleRunInput    map[string]any   `json:"exampleRunInput"`
	TaggedBuilds       map[string]any   `json:"taggedBuilds"`
	Readme             string           `json:"readme"`
	ReadmeMarkdown     string           `json:"readmeMarkdown"`
	InputSchema        map[string]any   `json:"inputSchema"`
	OutputSchema       map[string]any   `json:"outputSchema"`
	LatestBuild        map[string]any   `json:"latestBuild,omitempty"`
	EnrichmentError    string           `json:"enrichmentError,omitempty"`
}

// ActorVersionsResponse is /v2/acts/{id}/versions response.
type ActorVersionsResponse struct {
	Data struct {
		Total int              `json:"total"`
		Items []map[string]any `json:"items"`
	} `json:"data"`
}

// ActorBuildResponse is /v2/actor-builds/{buildId} response.
type ActorBuildResponse struct {
	Data map[string]any `json:"data"`
}

// CrawlStats tracks crawl progress.
type CrawlStats struct {
	ExpectedTotal  int
	IndexPages     int
	IndexedTotal   int64
	DetailQueued   int64
	DetailDone     int64
	DetailSuccess  int64
	DetailFailed   int64
	StartedAt      time.Time
	FinishedAt     time.Time
	CurrentRunUUID string
}
