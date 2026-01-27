package local

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local/engines"
)

// Adapter wraps MetaSearch to implement engine.Engine interface.
type Adapter struct {
	ms *MetaSearch
}

// NewAdapter creates a new Adapter wrapping a MetaSearch instance.
func NewAdapter(ms *MetaSearch) *Adapter {
	return &Adapter{ms: ms}
}

// NewAdapterWithConfig creates a new Adapter with the given configuration.
func NewAdapterWithConfig(config *Config) *Adapter {
	return &Adapter{ms: New(config)}
}

// NewAdapterWithDefaults creates a new Adapter with default configuration.
func NewAdapterWithDefaults() *Adapter {
	return &Adapter{ms: New(nil)}
}

// Search implements engine.Engine.
func (a *Adapter) Search(ctx context.Context, query string, opts engine.SearchOptions) (*engine.SearchResponse, error) {
	// Convert engine.SearchOptions to local.SearchOptions
	localOpts := a.convertSearchOptions(opts)

	// Perform search
	localResp, err := a.ms.Search(ctx, query, localOpts)
	if err != nil {
		return nil, err
	}

	// Convert local.SearchResponse to engine.SearchResponse
	return a.convertSearchResponse(localResp), nil
}

// Categories implements engine.Engine.
func (a *Adapter) Categories() []engine.Category {
	return []engine.Category{
		engine.CategoryGeneral,
		engine.CategoryImages,
		engine.CategoryVideos,
		engine.CategoryNews,
		engine.CategoryMusic,
		engine.CategoryFiles,
		engine.CategoryIT,
		engine.CategoryScience,
		engine.CategorySocial,
		engine.CategoryMaps,
	}
}

// Name implements engine.Engine.
func (a *Adapter) Name() string {
	return "local"
}

// Healthz checks if the engine is healthy.
func (a *Adapter) Healthz(ctx context.Context) error {
	// Local engine is always healthy since it's built-in
	return nil
}

// MetaSearch returns the underlying MetaSearch instance.
func (a *Adapter) MetaSearch() *MetaSearch {
	return a.ms
}

// convertSearchOptions converts engine.SearchOptions to local.SearchOptions.
func (a *Adapter) convertSearchOptions(opts engine.SearchOptions) SearchOptions {
	localOpts := SearchOptions{
		Page:    opts.Page,
		PerPage: opts.PerPage,
	}

	// Convert category
	if opts.Category != "" {
		localOpts.Categories = []Category{a.convertToLocalCategory(opts.Category)}
	}

	// Convert time range
	switch opts.TimeRange {
	case "day":
		localOpts.TimeRange = TimeRangeDay
	case "week":
		localOpts.TimeRange = TimeRangeWeek
	case "month":
		localOpts.TimeRange = TimeRangeMonth
	case "year":
		localOpts.TimeRange = TimeRangeYear
	}

	// Convert safe search
	switch opts.SafeSearch {
	case 0:
		localOpts.SafeSearch = SafeSearchOff
	case 1:
		localOpts.SafeSearch = SafeSearchModerate
	case 2:
		localOpts.SafeSearch = SafeSearchStrict
	}

	// Set language
	localOpts.Language = opts.Language
	localOpts.Locale = opts.Region

	return localOpts
}

// convertSearchResponse converts local.SearchResponse to engine.SearchResponse.
func (a *Adapter) convertSearchResponse(resp *SearchResponse) *engine.SearchResponse {
	engineResp := &engine.SearchResponse{
		Query:          resp.Query,
		TotalResults:   resp.TotalResults,
		SearchTimeMs:   resp.SearchTimeMs,
		Page:           resp.Page,
		PerPage:        resp.PerPage,
		Suggestions:    resp.Suggestions,
	}

	// Convert corrected query
	if len(resp.Corrections) > 0 {
		engineResp.CorrectedQuery = resp.Corrections[0]
	}

	// Convert results
	engineResp.Results = make([]engine.Result, len(resp.Results))
	for i, r := range resp.Results {
		engineResp.Results[i] = a.convertResult(r)
	}

	// Convert infoboxes
	engineResp.Infoboxes = make([]engine.Infobox, len(resp.Infoboxes))
	for i, ib := range resp.Infoboxes {
		engineResp.Infoboxes[i] = a.convertInfobox(ib)
	}

	return engineResp
}

// convertResult converts a local.Result to engine.Result.
func (a *Adapter) convertResult(r engines.Result) engine.Result {
	result := engine.Result{
		URL:         r.URL,
		Title:       r.Title,
		Content:     r.Content,
		Category:    a.convertToEngineCategory(r.Category),
		Engine:      r.Engine,
		Engines:     r.Engines,
		Score:       r.Score,
		PublishedAt: r.PublishedAt,

		// Image fields
		ThumbnailURL: r.ThumbnailURL,
		ImageURL:     r.ImageURL,
		ImgFormat:    r.ImgFormat,
		Source:       r.Source,
		Resolution:   r.Resolution,

		// Video fields
		Duration:  r.Duration,
		EmbedURL:  r.EmbedURL,
		IFrameSrc: r.IFrameSrc,

		// Music fields
		Artist: r.Artist,
		Album:  r.Album,
		Track:  r.Track,

		// File fields
		FileSize:   r.FileSize,
		MagnetLink: r.MagnetLink,
		Seed:       r.Seed,
		Leech:      r.Leech,

		// Science fields
		DOI:         r.DOI,
		ISSN:        r.ISSN,
		ISBN:        r.ISBN,
		Authors:     r.Authors,
		Publisher:   r.Publisher,
		Journal:     r.Journal,
		Type:        r.Type,
		AccessRight: r.AccessRight,

		// Map fields
		Latitude:    r.Latitude,
		Longitude:   r.Longitude,
		BoundingBox: r.BoundingBox,
		Geojson:     r.GeoJSON,
		OSMType:     r.OSMType,
		OSMID:       r.OSMID,
	}

	// Convert parsed URL to string array
	if r.ParsedURL != nil {
		result.ParsedURL = []string{
			r.ParsedURL.Scheme,
			r.ParsedURL.Host,
			r.ParsedURL.Path,
		}
	}

	// Convert address
	if r.Address != nil {
		result.Address = &engine.Address{
			Name:        r.Address.Name,
			Road:        r.Address.Road,
			Locality:    r.Address.Locality,
			PostalCode:  r.Address.PostalCode,
			Country:     r.Address.Country,
			CountryCode: r.Address.CountryCode,
			HouseNumber: r.Address.HouseNumber,
		}
	}

	return result
}

// convertInfobox converts a local.Infobox to engine.Infobox.
func (a *Adapter) convertInfobox(ib engines.Infobox) engine.Infobox {
	engineIb := engine.Infobox{
		ID:       ib.ID,
		Infobox:  ib.Title,
		Content:  ib.Content,
		ImageURL: ib.ImageURL,
		Engine:   ib.Engine,
	}

	// Convert URLs
	engineIb.URLs = make([]engine.InfoboxURL, len(ib.URLs))
	for i, u := range ib.URLs {
		engineIb.URLs[i] = engine.InfoboxURL{
			Title: u.Title,
			URL:   u.URL,
		}
	}

	// Convert attributes
	engineIb.Attributes = make([]engine.InfoboxAttribute, len(ib.Attributes))
	for i, attr := range ib.Attributes {
		engineIb.Attributes[i] = engine.InfoboxAttribute{
			Label: attr.Label,
			Value: attr.Value,
		}
	}

	return engineIb
}

// convertToLocalCategory converts engine.Category to local.Category.
func (a *Adapter) convertToLocalCategory(cat engine.Category) Category {
	switch cat {
	case engine.CategoryGeneral:
		return CategoryGeneral
	case engine.CategoryImages:
		return CategoryImages
	case engine.CategoryVideos:
		return CategoryVideos
	case engine.CategoryNews:
		return CategoryNews
	case engine.CategoryMusic:
		return CategoryMusic
	case engine.CategoryFiles:
		return CategoryFiles
	case engine.CategoryIT:
		return CategoryIT
	case engine.CategoryScience:
		return CategoryScience
	case engine.CategorySocial:
		return CategorySocial
	case engine.CategoryMaps:
		return CategoryMaps
	default:
		return CategoryGeneral
	}
}

// convertToEngineCategory converts local.Category to engine.Category.
func (a *Adapter) convertToEngineCategory(cat Category) engine.Category {
	switch cat {
	case CategoryGeneral, CategoryWeb:
		return engine.CategoryGeneral
	case CategoryImages:
		return engine.CategoryImages
	case CategoryVideos:
		return engine.CategoryVideos
	case CategoryNews:
		return engine.CategoryNews
	case CategoryMusic:
		return engine.CategoryMusic
	case CategoryFiles:
		return engine.CategoryFiles
	case CategoryIT:
		return engine.CategoryIT
	case CategoryScience:
		return engine.CategoryScience
	case CategorySocial:
		return engine.CategorySocial
	case CategoryMaps:
		return engine.CategoryMaps
	default:
		return engine.CategoryGeneral
	}
}
