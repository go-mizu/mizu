package local

import (
	"hash/fnv"
	"net/url"
	"sort"
	"sync"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local/engines"
)

// ResultContainer collects, aggregates and sorts results from multiple engines.
type ResultContainer struct {
	mu sync.RWMutex

	// Main results indexed by hash
	mainResults map[uint64]*engines.Result

	// Other result types
	infoboxes   []engines.Infobox
	suggestions map[string]struct{}
	answers     []engines.Answer
	corrections map[string]struct{}

	// Engine data
	engineData map[string]map[string]string

	// Metadata
	numberOfResults []int64
	timings         []EngineTiming
	unresponsive    []UnresponsiveEngine

	// State
	closed bool
	paging bool

	// Sorted results cache
	sortedResults []engines.Result
}

// NewResultContainer creates a new ResultContainer.
func NewResultContainer() *ResultContainer {
	return &ResultContainer{
		mainResults:     make(map[uint64]*engines.Result),
		infoboxes:       make([]engines.Infobox, 0),
		suggestions:     make(map[string]struct{}),
		answers:         make([]engines.Answer, 0),
		corrections:     make(map[string]struct{}),
		engineData:      make(map[string]map[string]string),
		numberOfResults: make([]int64, 0),
		timings:         make([]EngineTiming, 0),
		unresponsive:    make([]UnresponsiveEngine, 0),
	}
}

// Extend adds results from an engine to the container.
func (rc *ResultContainer) Extend(engineName string, results *engines.EngineResults) {
	if rc.closed || results == nil {
		return
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Add main results
	for i, result := range results.Results {
		result.Engine = engineName
		if result.Engines == nil {
			result.Engines = []string{engineName}
		}
		rc.addResult(&result, i+1)
	}

	// Add suggestions
	for _, s := range results.Suggestions {
		rc.suggestions[s] = struct{}{}
	}

	// Add corrections
	for _, c := range results.Corrections {
		rc.corrections[c] = struct{}{}
	}

	// Add answers
	for _, a := range results.Answers {
		rc.answers = append(rc.answers, a)
	}

	// Add infoboxes
	for _, ib := range results.Infoboxes {
		ib.Engine = engineName
		rc.mergeInfobox(ib)
	}

	// Add engine data
	if len(results.EngineData) > 0 {
		if rc.engineData[engineName] == nil {
			rc.engineData[engineName] = make(map[string]string)
		}
		for k, v := range results.EngineData {
			rc.engineData[engineName][k] = v
		}
	}
}

func (rc *ResultContainer) addResult(result *engines.Result, position int) {
	// Normalize URL
	if result.ParsedURL == nil && result.URL != "" {
		result.ParsedURL, _ = url.Parse(result.URL)
	}

	// Calculate hash
	result.Hash = rc.hashResult(result)

	// Check for existing result
	if existing, ok := rc.mainResults[result.Hash]; ok {
		// Merge with existing result
		rc.mergeResults(existing, result)
		existing.Positions = append(existing.Positions, position)
		return
	}

	// New result
	result.Positions = []int{position}
	rc.mainResults[result.Hash] = result
}

func (rc *ResultContainer) hashResult(result *engines.Result) uint64 {
	h := fnv.New64a()
	// Hash based on normalized URL
	if result.ParsedURL != nil {
		// Use host + path as key (ignoring query params and fragment)
		h.Write([]byte(result.ParsedURL.Host))
		h.Write([]byte(result.ParsedURL.Path))
	} else {
		h.Write([]byte(result.URL))
	}
	return h.Sum64()
}

func (rc *ResultContainer) mergeResults(existing, newResult *engines.Result) {
	// Use longer content
	if len(newResult.Content) > len(existing.Content) {
		existing.Content = newResult.Content
	}

	// Use longer title
	if len(newResult.Title) > len(existing.Title) {
		existing.Title = newResult.Title
	}

	// Add engine to list
	found := false
	for _, e := range existing.Engines {
		if e == newResult.Engine {
			found = true
			break
		}
	}
	if !found {
		existing.Engines = append(existing.Engines, newResult.Engine)
	}

	// Prefer HTTPS
	if existing.ParsedURL != nil && newResult.ParsedURL != nil {
		if !isSecure(existing.ParsedURL.Scheme) && isSecure(newResult.ParsedURL.Scheme) {
			existing.ParsedURL = newResult.ParsedURL
			existing.URL = newResult.URL
		}
	}

	// Merge optional fields
	if existing.ThumbnailURL == "" && newResult.ThumbnailURL != "" {
		existing.ThumbnailURL = newResult.ThumbnailURL
	}
	if existing.ImageURL == "" && newResult.ImageURL != "" {
		existing.ImageURL = newResult.ImageURL
	}
	if existing.Duration == "" && newResult.Duration != "" {
		existing.Duration = newResult.Duration
	}
	if existing.PublishedAt.IsZero() && !newResult.PublishedAt.IsZero() {
		existing.PublishedAt = newResult.PublishedAt
	}
}

func isSecure(scheme string) bool {
	return scheme == "https" || scheme == "ftps"
}

func (rc *ResultContainer) mergeInfobox(newIB engines.Infobox) {
	// Check for existing infobox with same ID
	for i, existing := range rc.infoboxes {
		if existing.ID != "" && existing.ID == newIB.ID {
			// Merge into existing
			if len(newIB.Content) > len(existing.Content) {
				rc.infoboxes[i].Content = newIB.Content
			}
			if existing.ImageURL == "" && newIB.ImageURL != "" {
				rc.infoboxes[i].ImageURL = newIB.ImageURL
			}
			// Merge URLs
			rc.infoboxes[i].URLs = append(rc.infoboxes[i].URLs, newIB.URLs...)
			// Merge attributes
			rc.infoboxes[i].Attributes = append(rc.infoboxes[i].Attributes, newIB.Attributes...)
			return
		}
	}
	// New infobox
	rc.infoboxes = append(rc.infoboxes, newIB)
}

// AddTiming adds timing information for an engine.
func (rc *ResultContainer) AddTiming(timing EngineTiming) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.timings = append(rc.timings, timing)
}

// AddUnresponsive adds an unresponsive engine.
func (rc *ResultContainer) AddUnresponsive(engine UnresponsiveEngine) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.unresponsive = append(rc.unresponsive, engine)
}

// AddNumberOfResults adds a result count from an engine.
func (rc *ResultContainer) AddNumberOfResults(count int64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.numberOfResults = append(rc.numberOfResults, count)
}

// SetPaging sets whether paging is available.
func (rc *ResultContainer) SetPaging(paging bool) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.paging = paging
}

// Close finalizes the result container and calculates scores.
func (rc *ResultContainer) Close() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.closed {
		return
	}
	rc.closed = true

	// Calculate scores for all results
	for _, result := range rc.mainResults {
		result.Score = rc.calculateScore(result)
	}
}

func (rc *ResultContainer) calculateScore(result *engines.Result) float64 {
	weight := 1.0 * float64(len(result.Engines))

	if result.Priority == engines.PriorityLow {
		return 0
	}

	score := 0.0
	for _, pos := range result.Positions {
		if result.Priority == engines.PriorityHigh {
			score += weight
		} else {
			score += weight / float64(pos)
		}
	}
	return score
}

// GetOrderedResults returns results sorted by score.
func (rc *ResultContainer) GetOrderedResults() []engines.Result {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if !rc.closed {
		rc.mu.Unlock()
		rc.Close()
		rc.mu.Lock()
	}

	if rc.sortedResults != nil {
		return rc.sortedResults
	}

	// Convert map to slice
	results := make([]engines.Result, 0, len(rc.mainResults))
	for _, r := range rc.mainResults {
		results = append(results, *r)
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	rc.sortedResults = results
	return results
}

// GetSuggestions returns all suggestions.
func (rc *ResultContainer) GetSuggestions() []string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	suggestions := make([]string, 0, len(rc.suggestions))
	for s := range rc.suggestions {
		suggestions = append(suggestions, s)
	}
	return suggestions
}

// GetCorrections returns all corrections.
func (rc *ResultContainer) GetCorrections() []string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	corrections := make([]string, 0, len(rc.corrections))
	for c := range rc.corrections {
		corrections = append(corrections, c)
	}
	return corrections
}

// GetAnswers returns all answers.
func (rc *ResultContainer) GetAnswers() []engines.Answer {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.answers
}

// GetInfoboxes returns all infoboxes.
func (rc *ResultContainer) GetInfoboxes() []engines.Infobox {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.infoboxes
}

// GetTimings returns all timing information.
func (rc *ResultContainer) GetTimings() []EngineTiming {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.timings
}

// GetUnresponsive returns all unresponsive engines.
func (rc *ResultContainer) GetUnresponsive() []UnresponsiveEngine {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.unresponsive
}

// GetEngineData returns engine data for subsequent requests.
func (rc *ResultContainer) GetEngineData() map[string]map[string]string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.engineData
}

// NumberOfResults returns the estimated total number of results.
func (rc *ResultContainer) NumberOfResults() int64 {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	if len(rc.numberOfResults) == 0 {
		return 0
	}

	var sum int64
	for _, n := range rc.numberOfResults {
		sum += n
	}
	avg := sum / int64(len(rc.numberOfResults))

	// Return 0 if average is less than actual results
	if avg < int64(len(rc.mainResults)) {
		return 0
	}
	return avg
}

// HasPaging returns whether paging is available.
func (rc *ResultContainer) HasPaging() bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.paging
}
