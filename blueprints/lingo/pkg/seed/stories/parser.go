package stories

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Parser parses story files in the duostories format
type Parser struct{}

// NewParser creates a new story parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses a story file content and returns StoryData
func (p *Parser) Parse(content string, fromLang, toLang string) (*StoryData, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	story := &StoryData{
		FromLanguage: fromLang,
		ToLanguage:   toLang,
		Characters:   []CharacterData{},
		Elements:     []ElementData{},
		CreatedAt:    time.Now(),
	}

	var currentSection string
	var currentElement *ElementData
	var elementPosition int
	characterMap := make(map[string]*CharacterData)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Check for section markers
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			// Save previous element if exists
			if currentElement != nil {
				story.Elements = append(story.Elements, *currentElement)
			}

			currentSection = strings.TrimPrefix(strings.TrimSuffix(trimmed, "]"), "[")
			currentElement = nil

			switch currentSection {
			case "HEADER":
				currentElement = &ElementData{
					Type:     ElementTypeHeader,
					Position: elementPosition,
				}
				elementPosition++
			case "LINE":
				currentElement = &ElementData{
					Type:     ElementTypeLine,
					Position: elementPosition,
				}
				elementPosition++
			case "MULTIPLE_CHOICE":
				currentElement = &ElementData{
					Type:     ElementTypeMultiChoice,
					Position: elementPosition,
					ChallengeData: &ChallengeInfo{
						CorrectAnswers: []string{},
						WrongAnswers:   []string{},
					},
				}
				elementPosition++
			case "SELECT_PHRASE":
				currentElement = &ElementData{
					Type:     ElementTypeSelectPhrase,
					Position: elementPosition,
					ChallengeData: &ChallengeInfo{
						CorrectAnswers: []string{},
						WrongAnswers:   []string{},
					},
				}
				elementPosition++
			case "ARRANGE":
				currentElement = &ElementData{
					Type:     ElementTypeArrange,
					Position: elementPosition,
					ChallengeData: &ChallengeInfo{
						Tokens: []Token{},
					},
				}
				elementPosition++
			case "MATCH":
				currentElement = &ElementData{
					Type:     ElementTypeMatch,
					Position: elementPosition,
					ChallengeData: &ChallengeInfo{
						Pairs: []Pair{},
					},
				}
				elementPosition++
			case "POINT_TO_PHRASE":
				currentElement = &ElementData{
					Type:     ElementTypePointPhrase,
					Position: elementPosition,
					ChallengeData: &ChallengeInfo{
						CorrectAnswers: []string{},
						WrongAnswers:   []string{},
					},
				}
				elementPosition++
			case "TAP_COMPLETE":
				currentElement = &ElementData{
					Type:     ElementTypeTapComplete,
					Position: elementPosition,
					ChallengeData: &ChallengeInfo{
						CorrectAnswers: []string{},
						WrongAnswers:   []string{},
					},
				}
				elementPosition++
			}
			continue
		}

		// Process line based on current section
		switch currentSection {
		case "DATA":
			p.parseDataLine(trimmed, story, characterMap)
		case "HEADER", "LINE":
			p.parseContentLine(trimmed, currentElement)
		case "MULTIPLE_CHOICE", "SELECT_PHRASE", "POINT_TO_PHRASE", "TAP_COMPLETE":
			p.parseChoiceLine(trimmed, currentElement)
		case "ARRANGE":
			p.parseArrangeLine(trimmed, currentElement)
		case "MATCH":
			p.parseMatchLine(trimmed, currentElement)
		}
	}

	// Save last element
	if currentElement != nil {
		story.Elements = append(story.Elements, *currentElement)
	}

	// Convert character map to slice
	for _, char := range characterMap {
		story.Characters = append(story.Characters, *char)
	}

	// Calculate duration based on elements
	story.DurationSeconds = p.estimateDuration(story)

	return story, scanner.Err()
}

// parseDataLine parses a line from the [DATA] section
func (p *Parser) parseDataLine(line string, story *StoryData, charMap map[string]*CharacterData) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	switch {
	case key == "fromLanguageName":
		story.Title = value
	case key == "icon":
		story.IconHash = value
	case key == "set":
		// Parse "set=<set_id>|<difficulty>"
		setParts := strings.Split(value, "|")
		if len(setParts) >= 1 {
			story.SetID, _ = strconv.Atoi(setParts[0])
		}
		if len(setParts) >= 2 {
			story.Difficulty, _ = strconv.Atoi(setParts[1])
		}
	case strings.HasPrefix(key, "icon_"):
		// Character avatar: icon_<Name>=<url>
		name := strings.TrimPrefix(key, "icon_")
		char := p.getOrCreateCharacter(charMap, name)
		char.AvatarURL = ImageBaseURL + value + ".svg"
	case strings.HasPrefix(key, "speaker_"):
		// Character voice: speaker_<Name>=<voice_id>
		name := strings.TrimPrefix(key, "speaker_")
		char := p.getOrCreateCharacter(charMap, name)
		char.VoiceID = value
	case key == "id":
		story.ExternalID = value
	}
}

// parseContentLine parses a line from content sections (HEADER, LINE)
func (p *Parser) parseContentLine(line string, elem *ElementData) {
	if elem == nil {
		return
	}

	switch {
	case strings.HasPrefix(line, ">"):
		// Main text or question
		text := strings.TrimSpace(strings.TrimPrefix(line, ">"))
		elem.Text = text
	case strings.HasPrefix(line, "~"):
		// Translation
		elem.Translation = strings.TrimSpace(strings.TrimPrefix(line, "~"))
	case strings.HasPrefix(line, "$"):
		// Audio reference: $<set>/<file>.mp3;<timing>
		p.parseAudioLine(line, elem)
	case strings.HasPrefix(line, "Speaker"):
		// Speaker line: Speaker<Name>: <text>
		p.parseSpeakerLine(line, elem)
	default:
		// Check for speaker pattern at start
		if match := regexp.MustCompile(`^([A-Za-z]+):\s*(.*)$`).FindStringSubmatch(line); match != nil {
			elem.SpeakerName = match[1]
			elem.Text = match[2]
		}
	}
}

// parseAudioLine parses audio reference and timing
func (p *Parser) parseAudioLine(line string, elem *ElementData) {
	// Format: $<set>/<file>.mp3;<start>,<duration>;...
	audio := strings.TrimPrefix(line, "$")
	parts := strings.SplitN(audio, ";", 2)

	if len(parts) >= 1 {
		// First part is the audio file path
		audioPath := strings.TrimSpace(parts[0])
		if !strings.HasPrefix(audioPath, "http") {
			elem.AudioURL = AudioBaseURL + audioPath
		} else {
			elem.AudioURL = audioPath
		}
	}

	if len(parts) >= 2 {
		// Parse timing data
		timingStr := parts[1]
		timingParts := strings.Split(timingStr, ";")
		for _, tp := range timingParts {
			if tp == "" {
				continue
			}
			values := strings.Split(tp, ",")
			if len(values) >= 2 {
				start, _ := strconv.Atoi(strings.TrimSpace(values[0]))
				duration, _ := strconv.Atoi(strings.TrimSpace(values[1]))
				elem.AudioTiming = append(elem.AudioTiming, AudioTimingData{
					Start:    start,
					Duration: duration,
				})
			}
		}
	}
}

// parseSpeakerLine parses a speaker dialogue line
func (p *Parser) parseSpeakerLine(line string, elem *ElementData) {
	// Format: Speaker<Name>: <text>
	match := regexp.MustCompile(`^Speaker([A-Za-z]+):\s*(.*)$`).FindStringSubmatch(line)
	if match != nil {
		elem.SpeakerName = match[1]
		elem.Text = match[2]
	}
}

// parseChoiceLine parses a line from choice sections (MULTIPLE_CHOICE, SELECT_PHRASE)
func (p *Parser) parseChoiceLine(line string, elem *ElementData) {
	if elem == nil || elem.ChallengeData == nil {
		return
	}

	switch {
	case strings.HasPrefix(line, ">"):
		// Question
		elem.ChallengeData.Question = strings.TrimSpace(strings.TrimPrefix(line, ">"))
	case strings.HasPrefix(line, "+"):
		// Correct answer
		answer := strings.TrimSpace(strings.TrimPrefix(line, "+"))
		elem.ChallengeData.CorrectAnswers = append(elem.ChallengeData.CorrectAnswers, answer)
	case strings.HasPrefix(line, "-"):
		// Wrong answer
		answer := strings.TrimSpace(strings.TrimPrefix(line, "-"))
		elem.ChallengeData.WrongAnswers = append(elem.ChallengeData.WrongAnswers, answer)
	case strings.HasPrefix(line, "~"):
		// Translation
		elem.Translation = strings.TrimSpace(strings.TrimPrefix(line, "~"))
	case strings.HasPrefix(line, "$"):
		// Audio
		p.parseAudioLine(line, elem)
	case strings.HasPrefix(line, "Speaker"):
		// Speaker line with select phrase
		p.parseSpeakerLine(line, elem)
		// Extract bracketed phrase as correct answer
		if matches := regexp.MustCompile(`\[([^\]]+)\]`).FindAllStringSubmatch(elem.Text, -1); matches != nil {
			for _, m := range matches {
				elem.ChallengeData.CorrectAnswers = append(elem.ChallengeData.CorrectAnswers, m[1])
			}
		}
	}
}

// parseArrangeLine parses a line from ARRANGE sections
func (p *Parser) parseArrangeLine(line string, elem *ElementData) {
	if elem == nil || elem.ChallengeData == nil {
		return
	}

	switch {
	case strings.HasPrefix(line, ">"):
		elem.ChallengeData.Question = strings.TrimSpace(strings.TrimPrefix(line, ">"))
	case strings.HasPrefix(line, "~"):
		elem.Translation = strings.TrimSpace(strings.TrimPrefix(line, "~"))
	case strings.HasPrefix(line, "$"):
		p.parseAudioLine(line, elem)
	case strings.HasPrefix(line, "Speaker"):
		// Parse tokens from speaker line
		p.parseSpeakerLine(line, elem)
		// Extract tokens: [(Word1) (Word2) ...]
		if matches := regexp.MustCompile(`\(([^)]+)\)`).FindAllStringSubmatch(elem.Text, -1); matches != nil {
			for i, m := range matches {
				token := Token{
					Text:  m[1],
					Order: i,
				}
				// Check for hint: Word~Hint format
				if parts := strings.SplitN(m[1], "~", 2); len(parts) == 2 {
					token.Text = parts[0]
					token.Hint = parts[1]
				}
				elem.ChallengeData.Tokens = append(elem.ChallengeData.Tokens, token)
			}
		}
	}
}

// parseMatchLine parses a line from MATCH sections
func (p *Parser) parseMatchLine(line string, elem *ElementData) {
	if elem == nil || elem.ChallengeData == nil {
		return
	}

	switch {
	case strings.HasPrefix(line, ">"):
		elem.ChallengeData.Question = strings.TrimSpace(strings.TrimPrefix(line, ">"))
	case strings.HasPrefix(line, "-"):
		// Parse pair: - word <> translation
		pairLine := strings.TrimSpace(strings.TrimPrefix(line, "-"))
		parts := strings.SplitN(pairLine, "<>", 2)
		if len(parts) == 2 {
			elem.ChallengeData.Pairs = append(elem.ChallengeData.Pairs, Pair{
				Left:  strings.TrimSpace(parts[0]),
				Right: strings.TrimSpace(parts[1]),
			})
		}
	}
}

// getOrCreateCharacter gets or creates a character in the map
func (p *Parser) getOrCreateCharacter(charMap map[string]*CharacterData, name string) *CharacterData {
	if char, ok := charMap[name]; ok {
		return char
	}
	char := &CharacterData{
		Name:        name,
		DisplayName: name,
	}
	charMap[name] = char
	return char
}

// estimateDuration estimates the story duration in seconds
func (p *Parser) estimateDuration(story *StoryData) int {
	duration := 0
	for _, elem := range story.Elements {
		switch elem.Type {
		case ElementTypeHeader:
			duration += 5
		case ElementTypeLine:
			// Estimate based on text length
			if elem.Text != "" {
				duration += len(elem.Text) / 20 // ~20 chars per second
			}
			if len(elem.AudioTiming) > 0 {
				// Use actual audio timing if available
				for _, t := range elem.AudioTiming {
					duration += t.Duration / 1000
				}
			}
		case ElementTypeMultiChoice, ElementTypeSelectPhrase:
			duration += 10
		case ElementTypeArrange:
			duration += 15
		case ElementTypeMatch:
			duration += 20
		default:
			duration += 5
		}
	}
	return duration
}

// ParseFromFile is a convenience method to parse a story file path
func (p *Parser) ParseFromFile(filePath string, fromLang, toLang string) (*StoryData, error) {
	// This would read the file and call Parse
	// For now, we expect content to be passed directly
	return nil, fmt.Errorf("not implemented - use Parse with content")
}
