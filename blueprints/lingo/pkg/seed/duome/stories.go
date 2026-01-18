package duome

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/html"
)

// StoryData represents a parsed story from Duome
type StoryData struct {
	ExternalID       string               `json:"external_id"`
	Title            string               `json:"title"`
	TitleTranslation string               `json:"title_translation,omitempty"`
	FromLanguage     string               `json:"from_language"`
	ToLanguage       string               `json:"to_language"`
	CEFRLevel        string               `json:"cefr_level,omitempty"`
	SentenceCount    int                  `json:"sentence_count"`
	ChoiceCount      int                  `json:"choice_count"`
	IllustrationURL  string               `json:"illustration_url,omitempty"`
	Characters       []StoryCharacterData `json:"characters"`
	Elements         []StoryElementData   `json:"elements"`
	AudioURLs        []string             `json:"audio_urls,omitempty"`
	FetchedAt        time.Time            `json:"fetched_at"`
}

// StoryCharacterData represents a character in a story
type StoryCharacterData struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}

// StoryElementData represents a single element in a story
type StoryElementData struct {
	Position    int                 `json:"position"`
	ElementType string              `json:"element_type"` // "line", "narration", "challenge", "header"
	Speaker     string              `json:"speaker,omitempty"`
	Text        string              `json:"text,omitempty"`
	Translation string              `json:"translation,omitempty"`
	AudioURL    string              `json:"audio_url,omitempty"`
	Challenge   *StoryChallengeData `json:"challenge,omitempty"`
}

// StoryChallengeData represents a challenge/question in a story
type StoryChallengeData struct {
	Type                string   `json:"type"` // "multiple_choice", "arrange", "select_phrase"
	Question            string   `json:"question,omitempty"`
	QuestionTranslation string   `json:"question_translation,omitempty"`
	Options             []string `json:"options,omitempty"`
	CorrectAnswer       string   `json:"correct_answer,omitempty"`
	CorrectIndex        int      `json:"correct_index,omitempty"`
	FeedbackCorrect     string   `json:"feedback_correct,omitempty"`
	FeedbackIncorrect   string   `json:"feedback_incorrect,omitempty"`
}

// StoryListItem represents a story in the list page
type StoryListItem struct {
	ExternalID       string `json:"external_id"`
	Title            string `json:"title"`
	TitleTranslation string `json:"title_translation,omitempty"`
	CEFRLevel        string `json:"cefr_level,omitempty"`
	IllustrationURL  string `json:"illustration_url,omitempty"`
	URL              string `json:"url"`
}

// StoryListData represents all stories for a language pair
type StoryListData struct {
	FromLanguage string          `json:"from_language"`
	ToLanguage   string          `json:"to_language"`
	Stories      []StoryListItem `json:"stories"`
	FetchedAt    time.Time       `json:"fetched_at"`
}

// StoriesPath returns the path for a stories list HTML file
func (d *Downloader) StoriesPath(pair LanguagePair) string {
	return filepath.Join(d.RawDir(), fmt.Sprintf("stories_%s_%s.html", pair.From, pair.To))
}

// StoryPath returns the path for a single story HTML file
func (d *Downloader) StoryPath(externalID string) string {
	return filepath.Join(d.RawDir(), "stories", fmt.Sprintf("%s.html", externalID))
}

// IsStoryDownloaded checks if a story file already exists
func (d *Downloader) IsStoryDownloaded(externalID string) bool {
	path := d.StoryPath(externalID)
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return true
	}
	return false
}

// DownloadStoriesList downloads the stories list page for a language pair
func (d *Downloader) DownloadStoriesList(ctx context.Context, pair LanguagePair) error {
	if err := d.ensureDirs(); err != nil {
		return err
	}

	path := d.StoriesPath(pair)

	// Skip if file already exists and has content
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return nil
	}

	// URL format: https://duome.eu/stories/en/es (from English to Spanish)
	url := fmt.Sprintf("https://duome.eu/stories/%s/%s", pair.From, pair.To)

	data, err := d.fetch(ctx, url)
	if err != nil {
		return fmt.Errorf("fetch stories list %s: %w", pair, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write stories list %s: %w", pair, err)
	}

	return nil
}

// DownloadStory downloads a single story page
func (d *Downloader) DownloadStory(ctx context.Context, externalID string) error {
	if err := d.ensureDirs(); err != nil {
		return err
	}

	// Ensure stories subdirectory exists
	storiesDir := filepath.Join(d.RawDir(), "stories")
	if err := os.MkdirAll(storiesDir, 0755); err != nil {
		return fmt.Errorf("create stories dir: %w", err)
	}

	path := d.StoryPath(externalID)

	// Skip if file already exists and has content
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return nil
	}

	url := fmt.Sprintf("https://duome.eu/stories/%s", externalID)

	data, err := d.fetch(ctx, url)
	if err != nil {
		return fmt.Errorf("fetch story %s: %w", externalID, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write story %s: %w", externalID, err)
	}

	return nil
}

// StoryDownloadResult holds the result of a story download
type StoryDownloadResult struct {
	ExternalID string
	Error      error
	Skipped    bool
}

// DownloadStoriesParallel downloads multiple stories in parallel
func (d *Downloader) DownloadStoriesParallel(ctx context.Context, storyIDs []string, workers int, progressFn func(done, total, skipped int, current string)) error {
	if workers <= 0 {
		workers = 5
	}

	// Ensure stories subdirectory exists
	storiesDir := filepath.Join(d.RawDir(), "stories")
	if err := os.MkdirAll(storiesDir, 0755); err != nil {
		return fmt.Errorf("create stories dir: %w", err)
	}

	total := len(storyIDs)
	var done int64
	var skipped int64

	// Create work channel
	work := make(chan string, len(storyIDs))
	for _, id := range storyIDs {
		work <- id
	}
	close(work)

	// Create worker pool
	var wg sync.WaitGroup
	errChan := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case externalID, ok := <-work:
					if !ok {
						return
					}

					// Check if already downloaded
					if d.IsStoryDownloaded(externalID) {
						atomic.AddInt64(&skipped, 1)
						atomic.AddInt64(&done, 1)
						if progressFn != nil {
							progressFn(int(atomic.LoadInt64(&done)), total, int(atomic.LoadInt64(&skipped)), externalID+" (cached)")
						}
						continue
					}

					// Download the story
					err := d.DownloadStory(ctx, externalID)
					atomic.AddInt64(&done, 1)

					if progressFn != nil {
						progressFn(int(atomic.LoadInt64(&done)), total, int(atomic.LoadInt64(&skipped)), externalID)
					}

					if err != nil {
						// Log but continue
						fmt.Printf("\n  Warning: failed to download %s: %v\n", externalID, err)
					}

					// Small delay to be nice to the server
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()
	}

	wg.Wait()
	close(errChan)

	return nil
}

// StoriesListJSONPath returns the path for a stories list JSON file
func (p *Parser) StoriesListJSONPath(pair LanguagePair) string {
	return filepath.Join(p.JSONDir(), fmt.Sprintf("stories_%s_%s.json", pair.From, pair.To))
}

// StoryJSONPath returns the path for a single story JSON file
func (p *Parser) StoryJSONPath(externalID string) string {
	return filepath.Join(p.JSONDir(), "stories", fmt.Sprintf("%s.json", externalID))
}

// StoriesPath returns the path for a stories list HTML file
func (p *Parser) StoriesPath(pair LanguagePair) string {
	return filepath.Join(p.RawDir(), fmt.Sprintf("stories_%s_%s.html", pair.From, pair.To))
}

// StoryPath returns the path for a single story HTML file
func (p *Parser) StoryPath(externalID string) string {
	return filepath.Join(p.RawDir(), "stories", fmt.Sprintf("%s.html", externalID))
}

// IsStoryParsed checks if a story JSON file already exists
func (p *Parser) IsStoryParsed(externalID string) bool {
	path := p.StoryJSONPath(externalID)
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return true
	}
	return false
}

// ParseStoriesList parses the stories list HTML to extract story metadata
func (p *Parser) ParseStoriesList(pair LanguagePair) (*StoryListData, error) {
	path := p.StoriesPath(pair)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	stories := extractStoryList(doc, pair)

	return &StoryListData{
		FromLanguage: pair.From,
		ToLanguage:   pair.To,
		Stories:      stories,
		FetchedAt:    time.Now(),
	}, nil
}

// extractStoryList extracts story items from the list page HTML
func extractStoryList(doc *html.Node, pair LanguagePair) []StoryListItem {
	var stories []StoryListItem
	seenIDs := make(map[string]bool)
	currentLevel := ""

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Look for level headers (h4 or similar with CEFR levels)
			if n.Data == "h4" || n.Data == "h3" || n.Data == "h5" {
				text := strings.TrimSpace(getTextContent(n))
				if isCEFRLevel(text) {
					currentLevel = text
				}
			}

			// Look for story links - they're usually in anchor tags with story URLs
			if n.Data == "a" {
				href := getAttr(n, "href")
				if strings.HasPrefix(href, "/stories/") {
					// This is an individual story link
					externalID := strings.TrimPrefix(href, "/stories/")

					// Skip if it's a language list link (like /stories/en or /stories/en/es)
					if len(externalID) <= 5 || !strings.Contains(externalID, "-") {
						goto next
					}

					// Skip duplicates
					if seenIDs[externalID] {
						goto next
					}
					seenIDs[externalID] = true

					story := StoryListItem{
						ExternalID: externalID,
						URL:        "https://duome.eu" + href,
						CEFRLevel:  currentLevel,
					}

					// Try to extract title from the link text or nearby elements
					title := strings.TrimSpace(getTextContent(n))
					if title != "" {
						story.Title = title
					}

					// Look for illustration URL in nearby img
					if img := findNearbyImg(n); img != nil {
						story.IllustrationURL = getAttr(img, "src")
					}

					stories = append(stories, story)
				}
			}
		}
	next:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return stories
}

// isCEFRLevel checks if text looks like a CEFR level
func isCEFRLevel(text string) bool {
	text = strings.ToUpper(strings.TrimSpace(text))
	levels := []string{"INTRO", "A1", "A2", "B1", "B2", "C1", "C2"}
	for _, level := range levels {
		if strings.HasPrefix(text, level) {
			return true
		}
	}
	return false
}

// findNearbyImg looks for an img element near the given node
func findNearbyImg(n *html.Node) *html.Node {
	// Check parent's children for img
	if n.Parent != nil {
		for c := n.Parent.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "img" {
				return c
			}
			// Check one level deeper
			for gc := c.FirstChild; gc != nil; gc = gc.NextSibling {
				if gc.Type == html.ElementNode && gc.Data == "img" {
					return gc
				}
			}
		}
	}
	return nil
}

// ParseStory parses a single story HTML file
func (p *Parser) ParseStory(externalID string, pair LanguagePair) (*StoryData, error) {
	path := p.StoryPath(externalID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	story := extractStoryContent(doc, externalID, pair)
	story.FetchedAt = time.Now()

	return story, nil
}

// extractStoryContent extracts all content from a story page
func extractStoryContent(doc *html.Node, externalID string, pair LanguagePair) *StoryData {
	story := &StoryData{
		ExternalID:   externalID,
		FromLanguage: pair.From,
		ToLanguage:   pair.To,
		Characters:   make([]StoryCharacterData, 0),
		Elements:     make([]StoryElementData, 0),
		AudioURLs:    make([]string, 0),
	}

	characterMap := make(map[string]bool)
	audioURLMap := make(map[string]bool)
	position := 0

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			class := getAttr(n, "class")

			// Extract audio URLs from data-src attributes
			dataSrc := getAttr(n, "data-src")
			if dataSrc != "" && (strings.Contains(dataSrc, "audio") || strings.Contains(dataSrc, ".mp3") || strings.Contains(dataSrc, ".wav") || strings.Contains(dataSrc, "duolingo")) {
				if !audioURLMap[dataSrc] {
					audioURLMap[dataSrc] = true
					story.AudioURLs = append(story.AudioURLs, dataSrc)
				}
			}

			// Also check src attribute for audio elements
			if n.Data == "audio" || n.Data == "source" {
				src := getAttr(n, "src")
				if src != "" && !audioURLMap[src] {
					audioURLMap[src] = true
					story.AudioURLs = append(story.AudioURLs, src)
				}
			}

			// Check for playback divs with audio data
			if strings.Contains(class, "playback") || strings.Contains(class, "sound") {
				dataSrc := getAttr(n, "data-src")
				if dataSrc != "" && !audioURLMap[dataSrc] {
					audioURLMap[dataSrc] = true
					story.AudioURLs = append(story.AudioURLs, dataSrc)
				}
			}

			// Extract title from h1
			if n.Data == "h1" {
				text := strings.TrimSpace(getTextContent(n))
				if text != "" && story.Title == "" {
					story.Title = text
				}
			}

			// Extract metadata (sentence count, choice count)
			if n.Data == "p" && strings.Contains(getTextContent(n), "sentences") {
				text := getTextContent(n)
				story.SentenceCount, story.ChoiceCount = parseStoryMetadata(text)
			}

			// Look for dialogue lines - typically in structured divs or paragraphs
			// Pattern: Speaker name followed by dialogue
			if n.Data == "p" || (n.Data == "div" && strings.Contains(class, "line")) {
				text := strings.TrimSpace(getTextContent(n))
				if text != "" {
					element := parseDialogueLine(text, position)
					if element != nil {
						// Try to find audio URL for this element
						audioURL := findElementAudio(n)
						if audioURL != "" {
							element.AudioURL = audioURL
							if !audioURLMap[audioURL] {
								audioURLMap[audioURL] = true
								story.AudioURLs = append(story.AudioURLs, audioURL)
							}
						}

						story.Elements = append(story.Elements, *element)
						position++

						// Track characters
						if element.Speaker != "" && !characterMap[element.Speaker] {
							characterMap[element.Speaker] = true
							story.Characters = append(story.Characters, StoryCharacterData{
								Name:        element.Speaker,
								DisplayName: element.Speaker,
							})
						}
					}
				}
			}

			// Look for challenges/questions (usually in list format)
			if n.Data == "ul" || n.Data == "ol" {
				challenge := parseChallenge(n, position)
				if challenge != nil {
					story.Elements = append(story.Elements, *challenge)
					position++
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return story
}

// findElementAudio looks for audio URL in or near an element
func findElementAudio(n *html.Node) string {
	// Check the element itself
	if dataSrc := getAttr(n, "data-src"); dataSrc != "" {
		return dataSrc
	}

	// Check children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			if dataSrc := getAttr(c, "data-src"); dataSrc != "" {
				return dataSrc
			}
			// Check for playback class
			class := getAttr(c, "class")
			if strings.Contains(class, "playback") || strings.Contains(class, "sound") {
				if dataSrc := getAttr(c, "data-src"); dataSrc != "" {
					return dataSrc
				}
			}
		}
	}

	// Check siblings
	if n.Parent != nil {
		for c := n.Parent.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				class := getAttr(c, "class")
				if strings.Contains(class, "playback") || strings.Contains(class, "sound") {
					if dataSrc := getAttr(c, "data-src"); dataSrc != "" {
						return dataSrc
					}
				}
			}
		}
	}

	return ""
}

// parseStoryMetadata extracts sentence and choice counts from text like "22 sentences, 14 choices"
func parseStoryMetadata(text string) (sentences, choices int) {
	re := regexp.MustCompile(`(\d+)\s*sentences?`)
	if matches := re.FindStringSubmatch(text); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &sentences)
	}
	re = regexp.MustCompile(`(\d+)\s*choices?`)
	if matches := re.FindStringSubmatch(text); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &choices)
	}
	return
}

// parseDialogueLine parses a dialogue line like "Lucy: Hello there!"
func parseDialogueLine(text string, position int) *StoryElementData {
	// Skip empty or very short lines
	if len(text) < 2 {
		return nil
	}

	element := &StoryElementData{
		Position:    position,
		ElementType: "line",
	}

	// Check for speaker pattern: "Name:" or "Name :" at the start
	colonIdx := strings.Index(text, ":")
	if colonIdx > 0 && colonIdx < 30 {
		potentialSpeaker := strings.TrimSpace(text[:colonIdx])
		// Validate it looks like a name (starts with capital, no special chars except space)
		if isValidSpeakerName(potentialSpeaker) {
			element.Speaker = potentialSpeaker
			element.Text = strings.TrimSpace(text[colonIdx+1:])
			element.ElementType = "line"
			return element
		}
	}

	// No speaker found, treat as narration
	element.ElementType = "narration"
	element.Text = text
	return element
}

// isValidSpeakerName checks if a string looks like a character name
func isValidSpeakerName(s string) bool {
	if len(s) == 0 || len(s) > 25 {
		return false
	}
	// First char should be uppercase letter or non-ASCII (for Japanese/Korean names)
	firstRune := []rune(s)[0]
	if !((firstRune >= 'A' && firstRune <= 'Z') || firstRune > 127) {
		return false
	}
	return true
}

// parseChallenge parses a challenge/question from a list element
func parseChallenge(ul *html.Node, position int) *StoryElementData {
	var options []string
	var question string

	// Look for preceding text as question/prompt
	if ul.PrevSibling != nil {
		question = strings.TrimSpace(getTextContent(ul.PrevSibling))
	}

	// Extract options from list items
	for li := ul.FirstChild; li != nil; li = li.NextSibling {
		if li.Type == html.ElementNode && li.Data == "li" {
			text := strings.TrimSpace(getTextContent(li))
			if text != "" {
				options = append(options, text)
			}
		}
	}

	if len(options) < 2 {
		return nil
	}

	// Set Text field so it displays even if challenge_data parsing fails
	displayText := question
	if displayText == "" && len(options) > 0 {
		displayText = "Choose the correct answer"
	}

	return &StoryElementData{
		Position:    position,
		ElementType: "multiple_choice", // Use specific type instead of generic "challenge"
		Text:        displayText,
		Challenge: &StoryChallengeData{
			Type:     "multiple_choice",
			Question: question,
			Options:  options,
		},
	}
}

// SaveStoryJSON saves a parsed story to JSON
func (p *Parser) SaveStoryJSON(story *StoryData) error {
	// Ensure stories JSON directory exists
	storiesDir := filepath.Join(p.JSONDir(), "stories")
	if err := os.MkdirAll(storiesDir, 0755); err != nil {
		return fmt.Errorf("create stories json dir: %w", err)
	}

	path := p.StoryJSONPath(story.ExternalID)
	data, err := json.MarshalIndent(story, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// SaveStoriesListJSON saves a parsed stories list to JSON
func (p *Parser) SaveStoriesListJSON(list *StoryListData, pair LanguagePair) error {
	if err := os.MkdirAll(p.JSONDir(), 0755); err != nil {
		return fmt.Errorf("create json dir: %w", err)
	}

	path := p.StoriesListJSONPath(pair)
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// LoadStoryJSON loads a parsed story from JSON
func (p *Parser) LoadStoryJSON(externalID string) (*StoryData, error) {
	path := p.StoryJSONPath(externalID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read json: %w", err)
	}

	var story StoryData
	if err := json.Unmarshal(data, &story); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	return &story, nil
}

// GetStoriesForPair returns the story external IDs available for a language pair
func (p *Parser) GetStoriesForPair(pair LanguagePair) ([]string, error) {
	path := p.StoriesListJSONPath(pair)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var list StoryListData
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}

	ids := make([]string, len(list.Stories))
	for i, s := range list.Stories {
		ids[i] = s.ExternalID
	}
	return ids, nil
}

// ParseStoriesParallel parses multiple stories in parallel
func (p *Parser) ParseStoriesParallel(storyIDs []string, pair LanguagePair, workers int, progressFn func(done, total int, current string)) ([]*StoryData, error) {
	if workers <= 0 {
		workers = 5
	}

	// Ensure stories JSON directory exists
	storiesDir := filepath.Join(p.JSONDir(), "stories")
	if err := os.MkdirAll(storiesDir, 0755); err != nil {
		return nil, fmt.Errorf("create stories json dir: %w", err)
	}

	total := len(storyIDs)
	var done int64

	// Results channel
	results := make(chan *StoryData, total)

	// Create work channel
	work := make(chan string, total)
	for _, id := range storyIDs {
		work <- id
	}
	close(work)

	// Create worker pool
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for externalID := range work {
				// Try to load from JSON cache first
				if p.IsStoryParsed(externalID) {
					story, err := p.LoadStoryJSON(externalID)
					if err == nil {
						atomic.AddInt64(&done, 1)
						if progressFn != nil {
							progressFn(int(atomic.LoadInt64(&done)), total, externalID+" (cached)")
						}
						results <- story
						continue
					}
				}

				// Parse from HTML
				story, err := p.ParseStory(externalID, pair)
				if err != nil {
					fmt.Printf("\n  Warning: failed to parse %s: %v\n", externalID, err)
					atomic.AddInt64(&done, 1)
					if progressFn != nil {
						progressFn(int(atomic.LoadInt64(&done)), total, externalID+" (error)")
					}
					continue
				}

				// Save to JSON
				if err := p.SaveStoryJSON(story); err != nil {
					fmt.Printf("\n  Warning: failed to save %s: %v\n", externalID, err)
				}

				atomic.AddInt64(&done, 1)
				if progressFn != nil {
					progressFn(int(atomic.LoadInt64(&done)), total, externalID)
				}
				results <- story
			}
		}()
	}

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	stories := make([]*StoryData, 0, total)
	for story := range results {
		stories = append(stories, story)
	}

	return stories, nil
}
