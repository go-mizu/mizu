package kaggle

import "time"

type Tag struct {
	Ref         string `json:"ref"`
	Name        string `json:"name"`
	Description string `json:"description"`
	FullPath    string `json:"fullPath"`
}

type DatasetFile struct {
	DatasetRef   string `json:"dataset_ref"`
	Name         string `json:"name"`
	TotalBytes   int64  `json:"total_bytes"`
	CreationDate string `json:"creation_date"`
}

type Dataset struct {
	ID                   int64         `json:"id"`
	Ref                  string        `json:"ref"`
	OwnerRef             string        `json:"owner_ref"`
	OwnerName            string        `json:"owner_name"`
	CreatorName          string        `json:"creator_name"`
	CreatorURL           string        `json:"creator_url"`
	Title                string        `json:"title"`
	Subtitle             string        `json:"subtitle"`
	Description          string        `json:"description"`
	URL                  string        `json:"url"`
	LicenseName          string        `json:"license_name"`
	ThumbnailImageURL    string        `json:"thumbnail_image_url"`
	DownloadCount        int64         `json:"download_count"`
	ViewCount            int64         `json:"view_count"`
	VoteCount            int64         `json:"vote_count"`
	KernelCount          int64         `json:"kernel_count"`
	TopicCount           int64         `json:"topic_count"`
	CurrentVersionNumber int           `json:"current_version_number"`
	UsabilityRating      float64       `json:"usability_rating"`
	TotalBytes           int64         `json:"total_bytes"`
	IsPrivate            bool          `json:"is_private"`
	IsFeatured           bool          `json:"is_featured"`
	LastUpdated          time.Time     `json:"last_updated"`
	Tags                 []Tag         `json:"tags"`
	Files                []DatasetFile `json:"files"`
	VersionsJSON         string        `json:"versions_json"`
	RawJSON              string        `json:"raw_json"`
	FetchedAt            time.Time     `json:"fetched_at"`
}

type ModelInstance struct {
	ModelRef               string `json:"model_ref"`
	InstanceID             int64  `json:"instance_id"`
	Slug                   string `json:"slug"`
	Framework              string `json:"framework"`
	FineTunable            bool   `json:"fine_tunable"`
	Overview               string `json:"overview"`
	Usage                  string `json:"usage"`
	DownloadURL            string `json:"download_url"`
	VersionID              int64  `json:"version_id"`
	VersionNumber          int    `json:"version_number"`
	URL                    string `json:"url"`
	LicenseName            string `json:"license_name"`
	ModelInstanceType      string `json:"model_instance_type"`
	ExternalBaseModelURL   string `json:"external_base_model_url"`
	TotalUncompressedBytes int64  `json:"total_uncompressed_bytes"`
	RawJSON                string `json:"raw_json"`
}

type Model struct {
	ID             int64           `json:"id"`
	Ref            string          `json:"ref"`
	OwnerRef       string          `json:"owner_ref"`
	Title          string          `json:"title"`
	Subtitle       string          `json:"subtitle"`
	Description    string          `json:"description"`
	Author         string          `json:"author"`
	AuthorImageURL string          `json:"author_image_url"`
	URL            string          `json:"url"`
	VoteCount      int64           `json:"vote_count"`
	UpdateTime     time.Time       `json:"update_time"`
	IsPrivate      bool            `json:"is_private"`
	Tags           []Tag           `json:"tags"`
	Instances      []ModelInstance `json:"instances"`
	RawJSON        string          `json:"raw_json"`
	FetchedAt      time.Time       `json:"fetched_at"`
}

type Competition struct {
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	ImageURL    string    `json:"image_url"`
	RawMetaJSON string    `json:"raw_meta_json"`
	FetchedAt   time.Time `json:"fetched_at"`
}

type Notebook struct {
	Ref         string    `json:"ref"`
	OwnerRef    string    `json:"owner_ref"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	ImageURL    string    `json:"image_url"`
	RawMetaJSON string    `json:"raw_meta_json"`
	FetchedAt   time.Time `json:"fetched_at"`
}

type Profile struct {
	Handle      string    `json:"handle"`
	DisplayName string    `json:"display_name"`
	Bio         string    `json:"bio"`
	URL         string    `json:"url"`
	ImageURL    string    `json:"image_url"`
	RawMetaJSON string    `json:"raw_meta_json"`
	FetchedAt   time.Time `json:"fetched_at"`
}

type PageMeta struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	OGTitle     string            `json:"og_title"`
	OGDesc      string            `json:"og_description"`
	OGImage     string            `json:"og_image"`
	URL         string            `json:"url"`
	Meta        map[string]string `json:"meta"`
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
	Datasets     int64
	Models       int64
	Competitions int64
	Notebooks    int64
	Profiles     int64
	DBSize       int64
}

type datasetAPIResponse struct {
	ID                   int64            `json:"id"`
	Ref                  string           `json:"ref"`
	CreatorName          string           `json:"creatorName"`
	CreatorURL           string           `json:"creatorUrl"`
	TotalBytes           int64            `json:"totalBytes"`
	URL                  string           `json:"url"`
	LicenseName          string           `json:"licenseName"`
	OwnerName            string           `json:"ownerName"`
	OwnerRef             string           `json:"ownerRef"`
	Title                string           `json:"title"`
	Subtitle             string           `json:"subtitle"`
	Description          string           `json:"description"`
	CurrentVersionNumber int              `json:"currentVersionNumber"`
	UsabilityRating      float64          `json:"usabilityRating"`
	ThumbnailImageURL    string           `json:"thumbnailImageUrl"`
	LastUpdated          string           `json:"lastUpdated"`
	DownloadCount        int64            `json:"downloadCount"`
	IsPrivate            bool             `json:"isPrivate"`
	IsFeatured           bool             `json:"isFeatured"`
	KernelCount          int64            `json:"kernelCount"`
	TopicCount           int64            `json:"topicCount"`
	ViewCount            int64            `json:"viewCount"`
	VoteCount            int64            `json:"voteCount"`
	Tags                 []Tag            `json:"tags"`
	Files                []datasetFileAPI `json:"files"`
	Versions             []any            `json:"versions"`
}

type datasetFileAPI struct {
	Name         string `json:"name"`
	TotalBytes   int64  `json:"totalBytes"`
	CreationDate string `json:"creationDate"`
}

type modelListResponse struct {
	Models        []modelAPI `json:"models"`
	NextPageToken string     `json:"nextPageToken"`
	TotalResults  int64      `json:"totalResults"`
}

type modelAPI struct {
	ID             int64              `json:"id"`
	Ref            string             `json:"ref"`
	Title          string             `json:"title"`
	Subtitle       string             `json:"subtitle"`
	Description    string             `json:"description"`
	Author         string             `json:"author"`
	AuthorImageURL string             `json:"authorImageUrl"`
	Slug           string             `json:"slug"`
	URL            string             `json:"url"`
	VoteCount      int64              `json:"voteCount"`
	UpdateTime     string             `json:"updateTime"`
	IsPrivate      bool               `json:"isPrivate"`
	Tags           []Tag              `json:"tags"`
	Instances      []modelInstanceAPI `json:"instances"`
}

type modelInstanceAPI struct {
	ID                     int64  `json:"id"`
	Slug                   string `json:"slug"`
	Framework              string `json:"framework"`
	FineTunable            bool   `json:"fineTunable"`
	Overview               string `json:"overview"`
	Usage                  string `json:"usage"`
	DownloadURL            string `json:"downloadUrl"`
	VersionID              int64  `json:"versionId"`
	VersionNumber          int    `json:"versionNumber"`
	URL                    string `json:"url"`
	LicenseName            string `json:"licenseName"`
	ModelInstanceType      string `json:"modelInstanceType"`
	ExternalBaseModelURL   string `json:"externalBaseModelUrl"`
	TotalUncompressedBytes int64  `json:"totalUncompressedBytes"`
}
