package local

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local/engines"
)

// MetaSearch is the main metasearch engine.
type MetaSearch struct {
	config     *Config
	registry   *Registry
	processors *ProcessorMap
	plugins    *PluginStorage
	answerers  *AnswererStorage
	cache      Cache
	httpClient *http.Client
}

// New creates a new MetaSearch engine with the given configuration.
func New(config *Config) *MetaSearch {
	if config == nil {
		config = DefaultConfig()
	}

	httpClient := &http.Client{
		Timeout: config.MaxRequestTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        config.MaxIdleConns,
			MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
			IdleConnTimeout:     config.IdleConnTimeout,
		},
	}

	ms := &MetaSearch{
		config:     config,
		registry:   NewRegistry(),
		processors: NewProcessorMap(),
		plugins:    NewPluginStorage(),
		answerers:  NewAnswererStorage(),
		httpClient: httpClient,
	}

	if config.CacheEnabled {
		ms.cache = NewMemoryCache(config.CacheTTL)
	}

	// Register built-in engines
	ms.registerBuiltinEngines()

	// Register built-in plugins
	ms.registerBuiltinPlugins()

	// Register built-in answerers
	ms.registerBuiltinAnswerers()

	return ms
}

// Search performs a search across all enabled engines.
func (ms *MetaSearch) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResponse, error) {
	startTime := time.Now()

	// Apply defaults
	if opts.Page == 0 {
		opts.Page = 1
	}
	if opts.PerPage == 0 {
		opts.PerPage = ms.config.DefaultPageSize
	}
	if opts.Language == "" {
		opts.Language = ms.config.DefaultLanguage
	}
	if opts.Locale == "" {
		opts.Locale = ms.config.DefaultLocale
	}
	if len(opts.Categories) == 0 {
		opts.Categories = ms.config.DefaultCategories
	}

	slog.Debug("search started", "query", query, "categories", opts.Categories, "page", opts.Page)

	// Create result container
	container := NewResultContainer()

	// Get answerers
	answers := ms.getAnswers(ctx, query)
	for _, a := range answers {
		container.answers = append(container.answers, a)
	}

	// Get engines to search
	engineRefs := ms.getEngineRefs(opts)
	if len(engineRefs) == 0 {
		slog.Warn("no engines available for search", "query", query, "categories", opts.Categories)
		return ms.buildResponse(query, container, startTime, opts), nil
	}

	// Run pre-search plugins
	search := &Search{
		Query:      query,
		Options:    opts,
		Container:  container,
		EngineRefs: engineRefs,
	}
	if !ms.plugins.PreSearch(ctx, search) {
		return ms.buildResponse(query, container, startTime, opts), nil
	}

	// Search engines in parallel
	ms.searchEngines(ctx, search)

	// Run post-search plugins
	ms.plugins.PostSearch(ctx, search)

	// Run on-result plugins for each result
	results := container.GetOrderedResults()
	filteredResults := make([]engines.Result, 0, len(results))
	for _, result := range results {
		if ms.plugins.OnResult(ctx, search, &result) {
			filteredResults = append(filteredResults, result)
		}
	}

	// Update container with filtered results
	container.sortedResults = filteredResults

	resp := ms.buildResponse(query, container, startTime, opts)

	// Log warnings for empty results or unresponsive engines
	if len(resp.Results) == 0 {
		slog.Warn("search returned no results",
			"query", query,
			"engines_queried", len(search.EngineRefs),
			"unresponsive_engines", len(resp.UnresponsiveEngines),
		)
	}
	if len(resp.UnresponsiveEngines) > 0 {
		for _, ue := range resp.UnresponsiveEngines {
			slog.Warn("engine unresponsive",
				"engine", ue.Engine,
				"error", ue.ErrorType,
				"suspended", ue.Suspended,
			)
		}
	}

	slog.Debug("search completed",
		"query", query,
		"results", len(resp.Results),
		"duration_ms", resp.SearchTimeMs,
	)

	return resp, nil
}

func (ms *MetaSearch) getEngineRefs(opts SearchOptions) []EngineRef {
	refs := make([]EngineRef, 0)

	// If specific engines are requested, use those
	if len(opts.Engines) > 0 {
		for _, name := range opts.Engines {
			if eng, ok := ms.registry.Get(name); ok {
				if !eng.Disabled() {
					cats := eng.Categories()
					cat := engines.CategoryGeneral
					if len(cats) > 0 {
						cat = cats[0]
					}
					refs = append(refs, EngineRef{Name: name, Category: cat})
				}
			}
		}
		return refs
	}

	// Get engines by category
	for _, cat := range opts.Categories {
		for _, eng := range ms.registry.GetByCategory(cat) {
			if !eng.Disabled() {
				refs = append(refs, EngineRef{Name: eng.Name(), Category: cat})
			}
		}
	}

	return refs
}

func (ms *MetaSearch) searchEngines(ctx context.Context, search *Search) {
	var wg sync.WaitGroup
	timeout := ms.config.RequestTimeout
	if search.Options.Timeout > 0 {
		timeout = search.Options.Timeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for _, ref := range search.EngineRefs {
		processor, ok := ms.processors.Get(ref.Name)
		if !ok {
			continue
		}

		// Skip suspended engines
		if processor.IsSuspended() {
			search.Container.AddUnresponsive(UnresponsiveEngine{
				Engine:    ref.Name,
				ErrorType: processor.SuspendReason(),
				Suspended: true,
			})
			continue
		}

		wg.Add(1)
		go func(ref EngineRef, proc *Processor) {
			defer wg.Done()
			ms.searchEngine(ctx, search, ref, proc)
		}(ref, processor)
	}

	wg.Wait()
}

func (ms *MetaSearch) searchEngine(ctx context.Context, search *Search, ref EngineRef, proc *Processor) {
	startTime := time.Now()

	// Get params for this engine
	params := proc.GetParams(search.Query, &search.Options, ref.Category)
	if params == nil {
		return
	}

	// Execute search
	results, err := proc.Search(ctx, search.Query, params)
	elapsed := time.Since(startTime)

	if err != nil {
		search.Container.AddUnresponsive(UnresponsiveEngine{
			Engine:    ref.Name,
			ErrorType: err.Error(),
			Suspended: false,
		})
		return
	}

	if results == nil {
		return
	}

	// Add results to container
	search.Container.Extend(ref.Name, results)

	// Add timing
	search.Container.AddTiming(EngineTiming{
		Engine: ref.Name,
		Total:  elapsed,
	})

	// Check paging support
	if proc.Engine().SupportsPaging() {
		search.Container.SetPaging(true)
	}
}

func (ms *MetaSearch) getAnswers(ctx context.Context, query string) []engines.Answer {
	return ms.answerers.Ask(ctx, query)
}

func (ms *MetaSearch) buildResponse(query string, container *ResultContainer, startTime time.Time, opts SearchOptions) *SearchResponse {
	container.Close()

	results := container.GetOrderedResults()
	suggestions := container.GetSuggestions()
	corrections := container.GetCorrections()
	answers := container.GetAnswers()
	infoboxes := container.GetInfoboxes()
	timings := container.GetTimings()
	unresponsive := container.GetUnresponsive()
	engineData := container.GetEngineData()
	totalResults := container.NumberOfResults()
	hasPaging := container.HasPaging()

	// Paginate results
	start := (opts.Page - 1) * opts.PerPage
	end := start + opts.PerPage
	if start > len(results) {
		start = len(results)
	}
	if end > len(results) {
		end = len(results)
	}
	pagedResults := results[start:end]

	// Convert infoboxes
	respInfoboxes := make([]Infobox, len(infoboxes))
	for i, ib := range infoboxes {
		respInfoboxes[i] = Infobox{
			ID:         ib.ID,
			Title:      ib.Title,
			Content:    ib.Content,
			ImageURL:   ib.ImageURL,
			URLs:       ib.URLs,
			Attributes: ib.Attributes,
			Engine:     ib.Engine,
		}
	}

	// Convert answers
	respAnswers := make([]Answer, len(answers))
	for i, a := range answers {
		respAnswers[i] = Answer{
			Answer: a.Answer,
			URL:    a.URL,
		}
	}

	return &SearchResponse{
		Query:               query,
		TotalResults:        totalResults,
		Results:             pagedResults,
		Suggestions:         suggestions,
		Corrections:         corrections,
		Answers:             respAnswers,
		Infoboxes:           respInfoboxes,
		SearchTimeMs:        float64(time.Since(startTime).Milliseconds()),
		Page:                opts.Page,
		PerPage:             opts.PerPage,
		Timings:             timings,
		UnresponsiveEngines: unresponsive,
		HasNextPage:         hasPaging && end < len(results),
		EngineData:          engineData,
	}
}

// Registry returns the engine registry.
func (ms *MetaSearch) Registry() *Registry {
	return ms.registry
}

// RegisterEngine registers a new engine.
func (ms *MetaSearch) RegisterEngine(eng engines.Engine) error {
	if err := ms.registry.Register(eng); err != nil {
		return err
	}
	ms.processors.Set(eng.Name(), NewProcessor(eng, ms.httpClient))
	return nil
}

// Search represents a search in progress.
type Search struct {
	Query      string
	Options    SearchOptions
	Container  *ResultContainer
	EngineRefs []EngineRef
}

func (ms *MetaSearch) registerBuiltinEngines() {
	// Register all built-in engines
	builtinEngines := []engines.Engine{
		// General/Web search
		engines.NewGoogle(),
		engines.NewBing(),
		engines.NewDuckDuckGo(),
		engines.NewWikipedia(),
		engines.NewWikidata(),
		engines.NewBrave(),
		engines.NewQwant(),

		// Image search
		engines.NewGoogleImages(),
		engines.NewBingImages(),
		engines.NewDuckDuckGoImages(),
		engines.NewQwantImages(),

		// Video search
		engines.NewYouTube(),

		// News search
		engines.NewBingNews(),
		engines.NewQwantNews(),

		// Social media
		engines.NewReddit(),

		// IT/Code
		engines.NewGitHub(),
		engines.NewGitHubCode(),

		// Science
		engines.NewArXiv(),
	}

	for _, eng := range builtinEngines {
		// Skip engines disabled in config
		engineDisabled := false
		for _, ec := range ms.config.Engines {
			if ec.Engine == eng.Name() && ec.Disabled {
				engineDisabled = true
				break
			}
		}
		if !engineDisabled {
			ms.registry.Register(eng)
			ms.processors.Set(eng.Name(), NewProcessor(eng, ms.httpClient))
		}
	}
}

func (ms *MetaSearch) registerBuiltinPlugins() {
	// Register default plugins
	ms.plugins.Register(NewTrackerURLRemoverPlugin())
	ms.plugins.Register(NewHostnameBlockerPlugin(nil))
	ms.plugins.Register(NewHashPlugin())
	ms.plugins.Register(NewHostnameReplacerPlugin(nil))
	ms.plugins.Register(NewSelfInfoPlugin())
	ms.plugins.Register(NewCalculatorPlugin())
	ms.plugins.Register(NewUnitConverterPlugin())
	ms.plugins.Register(NewOpenAccessDOIRewritePlugin())
	ms.plugins.Register(NewTimezonePlugin())

	// Activate plugins based on config
	for _, pc := range ms.config.Plugins {
		if pc.Active {
			ms.plugins.Enable(pc.ID)
		} else {
			ms.plugins.Disable(pc.ID)
		}
	}
}

func (ms *MetaSearch) registerBuiltinAnswerers() {
	// Register built-in answerers
	ms.answerers.Register(NewRandomAnswerer())
	ms.answerers.Register(NewHashAnswerer())
	ms.answerers.Register(NewDateTimeAnswerer())
	ms.answerers.Register(NewStatisticsAnswerer())
	ms.answerers.Register(NewColorAnswerer())
}
