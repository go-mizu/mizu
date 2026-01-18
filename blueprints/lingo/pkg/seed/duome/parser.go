package duome

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Parser parses downloaded HTML files into structured data
type Parser struct {
	baseDir  string
	progress ProgressCallback
}

// ParserOption configures a Parser
type ParserOption func(*Parser)

// WithParserProgress sets the progress callback
func WithParserProgress(cb ProgressCallback) ParserOption {
	return func(p *Parser) {
		p.progress = cb
	}
}

// NewParser creates a new Parser
func NewParser(baseDir string, opts ...ParserOption) *Parser {
	p := &Parser{
		baseDir: baseDir,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// RawDir returns the directory for raw HTML files
func (p *Parser) RawDir() string {
	return filepath.Join(p.baseDir, "raw")
}

// JSONDir returns the directory for parsed JSON files
func (p *Parser) JSONDir() string {
	return filepath.Join(p.baseDir, "json")
}

// VocabularyPath returns the path for a vocabulary HTML file
func (p *Parser) VocabularyPath(pair LanguagePair) string {
	return filepath.Join(p.RawDir(), fmt.Sprintf("vocabulary_%s_%s.html", pair.From, pair.To))
}

// TipsPath returns the path for a tips HTML file
func (p *Parser) TipsPath(pair LanguagePair) string {
	return filepath.Join(p.RawDir(), fmt.Sprintf("tips_%s_%s.html", pair.From, pair.To))
}

// VocabularyJSONPath returns the path for a vocabulary JSON file
func (p *Parser) VocabularyJSONPath(pair LanguagePair) string {
	return filepath.Join(p.JSONDir(), fmt.Sprintf("vocabulary_%s_%s.json", pair.From, pair.To))
}

// TipsJSONPath returns the path for a tips JSON file
func (p *Parser) TipsJSONPath(pair LanguagePair) string {
	return filepath.Join(p.JSONDir(), fmt.Sprintf("tips_%s_%s.json", pair.From, pair.To))
}

// ParseVocabulary parses a vocabulary HTML file
func (p *Parser) ParseVocabulary(pair LanguagePair) (*CourseData, error) {
	path := p.VocabularyPath(pair)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	// Extract vocabulary entries
	entries := extractVocabularyEntries(doc)

	// Group by skill
	skillMap := make(map[string]*Skill)
	skillOrder := make([]string, 0)

	for _, entry := range entries {
		skillName := entry.SkillName
		if skillName == "" {
			skillName = "Unknown"
		}

		skill, exists := skillMap[skillName]
		if !exists {
			skill = &Skill{
				Name:       skillName,
				Position:   len(skillOrder) + 1,
				Vocabulary: make([]VocabularyEntry, 0),
			}
			skillMap[skillName] = skill
			skillOrder = append(skillOrder, skillName)
		}
		skill.Vocabulary = append(skill.Vocabulary, entry)
	}

	// Build skill list in order
	skills := make([]Skill, 0, len(skillOrder))
	for _, name := range skillOrder {
		skills = append(skills, *skillMap[name])
	}

	return &CourseData{
		FromLanguage: pair.From,
		ToLanguage:   pair.To,
		Skills:       skills,
		TotalWords:   len(entries),
		FetchedAt:    time.Now(),
	}, nil
}

// extractVocabularyEntries extracts vocabulary entries from HTML
// Duome HTML structure:
// <div id="words"><ul class="plain list">
//
//	<li class="single"><div class="path-section-delimiter"><span>SkillName</span></div></li>
//	<li><span class="_blue wA">word</span> <span class="cCCC"> - [romaji]</span><span class="cCCC wT"> - trans1, trans2</span></li>
//
// </ul></div>
func extractVocabularyEntries(doc *html.Node) []VocabularyEntry {
	entries := make([]VocabularyEntry, 0)
	currentSkill := "Unknown"

	// Find the #words div first
	var wordsDiv *html.Node
	var findWordsDiv func(*html.Node)
	findWordsDiv = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == "words" {
					wordsDiv = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findWordsDiv(c)
		}
	}
	findWordsDiv(doc)

	if wordsDiv == nil {
		return entries
	}

	// Find the ul element
	var ul *html.Node
	for c := wordsDiv.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "ul" {
			ul = c
			break
		}
	}
	if ul == nil {
		return entries
	}

	// Process each li
	for li := ul.FirstChild; li != nil; li = li.NextSibling {
		if li.Type != html.ElementNode || li.Data != "li" {
			continue
		}

		// Check if this is a skill delimiter
		class := getAttr(li, "class")
		if strings.Contains(class, "single") {
			// Look for path-section-delimiter
			var delimSpan *html.Node
			var findDelim func(*html.Node)
			findDelim = func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == "span" {
					parent := n.Parent
					if parent != nil && parent.Type == html.ElementNode {
						parentClass := getAttr(parent, "class")
						if strings.Contains(parentClass, "path-section-delimiter") {
							delimSpan = n
							return
						}
					}
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					findDelim(c)
				}
			}
			findDelim(li)

			if delimSpan != nil {
				skillName := strings.TrimSpace(getTextContent(delimSpan))
				if skillName != "" {
					currentSkill = skillName
				}
			}
			continue
		}

		// This is a vocabulary entry
		entry := parseVocabularyLiDuome(li, currentSkill)
		if entry != nil && entry.Word != "" {
			entries = append(entries, *entry)
		}
	}

	return entries
}

// parseVocabularyLiDuome parses a Duome vocabulary list item
// Structure: <span class="_blue wA">word</span> <span class="cCCC"> - [romaji]</span><span class="cCCC wT"> - translations</span>
func parseVocabularyLiDuome(li *html.Node, skillName string) *VocabularyEntry {
	entry := &VocabularyEntry{
		SkillName: skillName,
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" {
			class := getAttr(n, "class")
			text := strings.TrimSpace(getTextContent(n))

			// Word: <span class="_blue wA">word</span>
			if strings.Contains(class, "wA") && strings.Contains(class, "_blue") {
				entry.Word = text
			} else if strings.Contains(class, "wT") {
				// Translations: <span class="cCCC wT"> - trans1, trans2</span>
				text = strings.TrimPrefix(text, "-")
				text = strings.TrimSpace(text)
				if text != "" {
					translations := strings.Split(text, ",")
					for i, t := range translations {
						translations[i] = strings.TrimSpace(t)
					}
					entry.Translations = translations
				}
			} else if strings.Contains(class, "cCCC") && !strings.Contains(class, "wT") {
				// Romanization: <span class="cCCC"> - [romaji]</span>
				// Extract romanization from brackets
				if strings.Contains(text, "[") && strings.Contains(text, "]") {
					start := strings.Index(text, "[")
					end := strings.Index(text, "]")
					if start < end {
						entry.Romanization = text[start+1 : end]
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(li)

	return entry
}

// parseVocabularyLi parses a single vocabulary list item
func parseVocabularyLi(li *html.Node) *VocabularyEntry {
	entry := &VocabularyEntry{}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			class := getAttr(n, "class")
			switch {
			case strings.Contains(class, "wN"):
				// Word name
				entry.Word = strings.TrimSpace(getTextContent(n))
			case strings.Contains(class, "wA"):
				// Romanization (in brackets like [ocha])
				text := strings.TrimSpace(getTextContent(n))
				// Remove brackets
				text = strings.TrimPrefix(text, "[")
				text = strings.TrimSuffix(text, "]")
				entry.Romanization = text
			case strings.Contains(class, "wG"):
				// Translations (comma-separated)
				text := strings.TrimSpace(getTextContent(n))
				if text != "" {
					translations := strings.Split(text, ",")
					for i, t := range translations {
						translations[i] = strings.TrimSpace(t)
					}
					entry.Translations = translations
				}
			case strings.Contains(class, "wT"):
				// Skill tag
				entry.SkillName = strings.TrimSpace(getTextContent(n))
			case strings.Contains(class, "wP"):
				// Part of speech
				text := strings.TrimSpace(getTextContent(n))
				// Clean up separators
				text = strings.Trim(text, " -–")
				if text != "" {
					entry.POS = text
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(li)

	return entry
}

// ParseTips parses a tips HTML file
func (p *Parser) ParseTips(pair LanguagePair) (*TipsData, error) {
	path := p.TipsPath(pair)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	// Extract tips sections
	skills := extractTipsSections(doc)

	return &TipsData{
		FromLanguage: pair.From,
		ToLanguage:   pair.To,
		Skills:       skills,
		FetchedAt:    time.Now(),
	}, nil
}

// extractTipsSections extracts tips sections from HTML
func extractTipsSections(doc *html.Node) map[string]*SkillTips {
	skills := make(map[string]*SkillTips)

	// Find all h4 elements (skill headers)
	var currentSkill *SkillTips
	var contentBuilder strings.Builder

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h4":
				// Save previous skill if exists
				if currentSkill != nil {
					currentSkill.Content = strings.TrimSpace(contentBuilder.String())
					skills[currentSkill.SkillName] = currentSkill
					contentBuilder.Reset()
				}

				// Start new skill
				text := getTextContent(n)
				skillName := parseSkillHeader(text)
				if skillName != "" {
					currentSkill = &SkillTips{
						SkillName: skillName,
						Tables:    make([]Table, 0),
						Examples:  make([]Example, 0),
					}
				}

			case "table":
				if currentSkill != nil {
					table := parseTable(n)
					if len(table.Rows) > 0 {
						currentSkill.Tables = append(currentSkill.Tables, table)
					}
				}

			case "p":
				if currentSkill != nil {
					text := strings.TrimSpace(getTextContent(n))
					if text != "" {
						contentBuilder.WriteString(text)
						contentBuilder.WriteString("\n\n")
					}
				}

			case "ul", "ol":
				if currentSkill != nil {
					listText := parseList(n)
					if listText != "" {
						contentBuilder.WriteString(listText)
						contentBuilder.WriteString("\n")
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// Save last skill
	if currentSkill != nil {
		currentSkill.Content = strings.TrimSpace(contentBuilder.String())
		skills[currentSkill.SkillName] = currentSkill
	}

	return skills
}

// parseSkillHeader extracts skill name from header text like "Basics · 11 · 2024-11-08"
func parseSkillHeader(text string) string {
	// Remove leading/trailing whitespace
	text = strings.TrimSpace(text)

	// Split by common delimiters
	parts := strings.Split(text, "·")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}

	// Try splitting by numbers
	re := regexp.MustCompile(`^([A-Za-z\s]+)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return text
}

// parseTable extracts a table from HTML
func parseTable(table *html.Node) Table {
	t := Table{
		Headers: make([]string, 0),
		Rows:    make([][]string, 0),
	}

	var walk func(*html.Node)
	inHeader := false

	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "thead":
				inHeader = true
			case "tbody":
				inHeader = false
			case "tr":
				if inHeader || len(t.Headers) == 0 {
					// This might be header row
					row := extractTableRow(n)
					if len(row) > 0 {
						if inHeader || isHeaderRow(n) {
							t.Headers = row
						} else {
							t.Rows = append(t.Rows, row)
						}
					}
				} else {
					row := extractTableRow(n)
					if len(row) > 0 {
						t.Rows = append(t.Rows, row)
					}
				}
				return // Don't recurse into tr
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(table)

	return t
}

// extractTableRow extracts cells from a table row
func extractTableRow(tr *html.Node) []string {
	cells := make([]string, 0)
	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
			cells = append(cells, strings.TrimSpace(getTextContent(c)))
		}
	}
	return cells
}

// isHeaderRow checks if a row contains th elements
func isHeaderRow(tr *html.Node) bool {
	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "th" {
			return true
		}
	}
	return false
}

// parseList extracts text from a list
func parseList(list *html.Node) string {
	var items []string
	for c := list.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "li" {
			text := strings.TrimSpace(getTextContent(c))
			if text != "" {
				items = append(items, "- "+text)
			}
		}
	}
	return strings.Join(items, "\n")
}

// getAttr gets an attribute value from a node
func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// getTextContent gets all text content from a node
func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(getTextContent(c))
	}
	return text.String()
}

// ParseAndSaveVocabulary parses vocabulary HTML and saves to JSON
func (p *Parser) ParseAndSaveVocabulary(pair LanguagePair) (*CourseData, error) {
	data, err := p.ParseVocabulary(pair)
	if err != nil {
		return nil, err
	}

	// Ensure JSON directory exists
	if err := os.MkdirAll(p.JSONDir(), 0755); err != nil {
		return nil, fmt.Errorf("create json dir: %w", err)
	}

	// Save to JSON
	jsonPath := p.VocabularyJSONPath(pair)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}

	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return nil, fmt.Errorf("write json: %w", err)
	}

	return data, nil
}

// ParseAndSaveTips parses tips HTML and saves to JSON
func (p *Parser) ParseAndSaveTips(pair LanguagePair) (*TipsData, error) {
	data, err := p.ParseTips(pair)
	if err != nil {
		return nil, err
	}

	// Ensure JSON directory exists
	if err := os.MkdirAll(p.JSONDir(), 0755); err != nil {
		return nil, fmt.Errorf("create json dir: %w", err)
	}

	// Save to JSON
	jsonPath := p.TipsJSONPath(pair)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}

	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return nil, fmt.Errorf("write json: %w", err)
	}

	return data, nil
}

// ParsePair parses both vocabulary and tips for a language pair
func (p *Parser) ParsePair(pair LanguagePair) (*CourseData, *TipsData, error) {
	vocab, err := p.ParseAndSaveVocabulary(pair)
	if err != nil {
		return nil, nil, fmt.Errorf("parse vocabulary: %w", err)
	}

	tips, err := p.ParseAndSaveTips(pair)
	if err != nil {
		// Tips might not exist for all languages, just log warning
		fmt.Printf("Warning: no tips found for %s: %v\n", pair, err)
		tips = &TipsData{
			FromLanguage: pair.From,
			ToLanguage:   pair.To,
			Skills:       make(map[string]*SkillTips),
			FetchedAt:    time.Now(),
		}
	}

	// Merge tips into vocabulary
	for i, skill := range vocab.Skills {
		if tip, ok := tips.Skills[skill.Name]; ok {
			vocab.Skills[i].Tips = tip
		}
	}

	return vocab, tips, nil
}

// ParseAll parses all downloaded language pairs
func (p *Parser) ParseAll(pairs []LanguagePair) (map[string]*CourseData, error) {
	results := make(map[string]*CourseData)
	total := len(pairs)

	for i, pair := range pairs {
		if p.progress != nil {
			p.progress(i+1, total, fmt.Sprintf("Parsing %s", pair))
		}

		// Check if vocabulary file exists
		vocabPath := p.VocabularyPath(pair)
		if _, err := os.Stat(vocabPath); os.IsNotExist(err) {
			fmt.Printf("Warning: vocabulary file not found for %s, skipping\n", pair)
			continue
		}

		vocab, _, err := p.ParsePair(pair)
		if err != nil {
			fmt.Printf("Warning: failed to parse %s: %v\n", pair, err)
			continue
		}

		results[pair.String()] = vocab
	}

	return results, nil
}

// LoadVocabularyJSON loads parsed vocabulary from JSON file
func (p *Parser) LoadVocabularyJSON(pair LanguagePair) (*CourseData, error) {
	path := p.VocabularyJSONPath(pair)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read json: %w", err)
	}

	var course CourseData
	if err := json.Unmarshal(data, &course); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	return &course, nil
}

// LoadTipsJSON loads parsed tips from JSON file
func (p *Parser) LoadTipsJSON(pair LanguagePair) (*TipsData, error) {
	path := p.TipsJSONPath(pair)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read json: %w", err)
	}

	var tips TipsData
	if err := json.Unmarshal(data, &tips); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	return &tips, nil
}
