package local

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local/engines"
)

// Answerer provides instant answers.
type Answerer interface {
	// Keywords that trigger this answerer.
	Keywords() []string

	// Info returns answerer metadata.
	Info() AnswererInfo

	// Answer returns instant answers for the query.
	Answer(ctx context.Context, query string) []engines.Answer
}

// AnswererInfo contains answerer metadata.
type AnswererInfo struct {
	ID          string
	Name        string
	Description string
	Keywords    []string
	Examples    []string
}

// AnswererStorage manages answerers.
type AnswererStorage struct {
	mu        sync.RWMutex
	answerers []Answerer
	keywords  map[string][]Answerer
}

// NewAnswererStorage creates a new AnswererStorage.
func NewAnswererStorage() *AnswererStorage {
	return &AnswererStorage{
		answerers: make([]Answerer, 0),
		keywords:  make(map[string][]Answerer),
	}
}

// Register registers an answerer.
func (as *AnswererStorage) Register(answerer Answerer) {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.answerers = append(as.answerers, answerer)

	for _, keyword := range answerer.Keywords() {
		kw := strings.ToLower(keyword)
		as.keywords[kw] = append(as.keywords[kw], answerer)
	}
}

// Ask asks all relevant answerers for answers.
func (as *AnswererStorage) Ask(ctx context.Context, query string) []engines.Answer {
	as.mu.RLock()
	defer as.mu.RUnlock()

	answers := make([]engines.Answer, 0)

	// Find answerers by keyword
	words := strings.Fields(strings.ToLower(query))
	askedAnswerers := make(map[Answerer]struct{})

	for _, word := range words {
		if answerers, ok := as.keywords[word]; ok {
			for _, a := range answerers {
				if _, asked := askedAnswerers[a]; !asked {
					askedAnswerers[a] = struct{}{}
					answers = append(answers, a.Answer(ctx, query)...)
				}
			}
		}
	}

	// Also try all answerers that might match without keywords
	for _, a := range as.answerers {
		if _, asked := askedAnswerers[a]; !asked {
			resp := a.Answer(ctx, query)
			if len(resp) > 0 {
				answers = append(answers, resp...)
			}
		}
	}

	return answers
}

// RandomAnswerer provides random number/string generation.
type RandomAnswerer struct {
	info     AnswererInfo
	patterns []*randomPattern
}

type randomPattern struct {
	re       *regexp.Regexp
	generate func(matches []string) string
}

// NewRandomAnswerer creates a new RandomAnswerer.
func NewRandomAnswerer() *RandomAnswerer {
	return &RandomAnswerer{
		info: AnswererInfo{
			ID:          "random",
			Name:        "Random Generator",
			Description: "Generate random numbers and strings",
			Keywords:    []string{"random", "uuid", "password"},
			Examples: []string{
				"random number 1 100",
				"random string 16",
				"uuid",
				"password 20",
			},
		},
		patterns: []*randomPattern{
			{
				re: regexp.MustCompile(`(?i)^random\s+number\s+(\d+)\s+(\d+)$`),
				generate: func(matches []string) string {
					min, max := 0, 100
					fmt.Sscanf(matches[1], "%d", &min)
					fmt.Sscanf(matches[2], "%d", &max)
					if min >= max {
						return ""
					}
					n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
					return fmt.Sprintf("%d", min+int(n.Int64()))
				},
			},
			{
				re: regexp.MustCompile(`(?i)^random\s+string\s+(\d+)$`),
				generate: func(matches []string) string {
					length := 16
					fmt.Sscanf(matches[1], "%d", &length)
					if length > 1000 {
						length = 1000
					}
					const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
					result := make([]byte, length)
					for i := range result {
						n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
						result[i] = charset[n.Int64()]
					}
					return string(result)
				},
			},
			{
				re: regexp.MustCompile(`(?i)^uuid$`),
				generate: func(matches []string) string {
					uuid := make([]byte, 16)
					rand.Read(uuid)
					uuid[6] = (uuid[6] & 0x0f) | 0x40
					uuid[8] = (uuid[8] & 0x3f) | 0x80
					return fmt.Sprintf("%x-%x-%x-%x-%x",
						uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
				},
			},
			{
				re: regexp.MustCompile(`(?i)^password\s*(\d*)$`),
				generate: func(matches []string) string {
					length := 20
					if matches[1] != "" {
						fmt.Sscanf(matches[1], "%d", &length)
					}
					if length > 100 {
						length = 100
					}
					const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
					result := make([]byte, length)
					for i := range result {
						n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
						result[i] = charset[n.Int64()]
					}
					return string(result)
				},
			},
		},
	}
}

func (a *RandomAnswerer) Keywords() []string { return a.info.Keywords }
func (a *RandomAnswerer) Info() AnswererInfo { return a.info }

func (a *RandomAnswerer) Answer(ctx context.Context, query string) []engines.Answer {
	query = strings.TrimSpace(query)

	for _, pattern := range a.patterns {
		matches := pattern.re.FindStringSubmatch(query)
		if len(matches) > 0 {
			result := pattern.generate(matches)
			if result != "" {
				return []engines.Answer{{Answer: result}}
			}
		}
	}

	return nil
}

// HashAnswerer provides hash calculation.
type HashAnswerer struct {
	info     AnswererInfo
	patterns []*hashAnswerPattern
}

type hashAnswerPattern struct {
	re       *regexp.Regexp
	hashFunc func(string) string
	name     string
}

// NewHashAnswerer creates a new HashAnswerer.
func NewHashAnswerer() *HashAnswerer {
	return &HashAnswerer{
		info: AnswererInfo{
			ID:          "hash",
			Name:        "Hash Calculator",
			Description: "Calculate hashes of strings",
			Keywords:    []string{"md5", "sha1", "sha256", "sha512", "hash"},
			Examples: []string{
				"md5 hello",
				"sha256 hello world",
			},
		},
		patterns: []*hashAnswerPattern{
			{
				re:       regexp.MustCompile(`(?i)^md5\s+(.+)$`),
				hashFunc: func(s string) string { h := md5.Sum([]byte(s)); return hex.EncodeToString(h[:]) },
				name:     "MD5",
			},
			{
				re:       regexp.MustCompile(`(?i)^sha1\s+(.+)$`),
				hashFunc: func(s string) string { h := sha1.Sum([]byte(s)); return hex.EncodeToString(h[:]) },
				name:     "SHA1",
			},
			{
				re:       regexp.MustCompile(`(?i)^sha256\s+(.+)$`),
				hashFunc: func(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) },
				name:     "SHA256",
			},
			{
				re:       regexp.MustCompile(`(?i)^sha512\s+(.+)$`),
				hashFunc: func(s string) string { h := sha512.Sum512([]byte(s)); return hex.EncodeToString(h[:]) },
				name:     "SHA512",
			},
		},
	}
}

func (a *HashAnswerer) Keywords() []string { return a.info.Keywords }
func (a *HashAnswerer) Info() AnswererInfo { return a.info }

func (a *HashAnswerer) Answer(ctx context.Context, query string) []engines.Answer {
	query = strings.TrimSpace(query)

	for _, pattern := range a.patterns {
		matches := pattern.re.FindStringSubmatch(query)
		if len(matches) >= 2 {
			input := matches[1]
			hash := pattern.hashFunc(input)
			return []engines.Answer{{
				Answer: pattern.name + ": " + hash,
			}}
		}
	}

	return nil
}

// DateTimeAnswerer provides current date/time.
type DateTimeAnswerer struct {
	info AnswererInfo
}

// NewDateTimeAnswerer creates a new DateTimeAnswerer.
func NewDateTimeAnswerer() *DateTimeAnswerer {
	return &DateTimeAnswerer{
		info: AnswererInfo{
			ID:          "datetime",
			Name:        "Date & Time",
			Description: "Show current date and time",
			Keywords:    []string{"time", "date", "now", "today"},
			Examples: []string{
				"what time is it",
				"current date",
				"today",
			},
		},
	}
}

func (a *DateTimeAnswerer) Keywords() []string { return a.info.Keywords }
func (a *DateTimeAnswerer) Info() AnswererInfo { return a.info }

func (a *DateTimeAnswerer) Answer(ctx context.Context, query string) []engines.Answer {
	query = strings.ToLower(strings.TrimSpace(query))

	patterns := []string{
		"time",
		"what time",
		"current time",
		"date",
		"what date",
		"current date",
		"today",
		"now",
	}

	for _, pattern := range patterns {
		if strings.Contains(query, pattern) {
			now := time.Now()
			return []engines.Answer{{
				Answer: now.Format("Monday, January 2, 2006 3:04 PM MST"),
			}}
		}
	}

	return nil
}

// StatisticsAnswerer provides statistical calculations.
type StatisticsAnswerer struct {
	info     AnswererInfo
	patterns []*statsPattern
}

type statsPattern struct {
	re      *regexp.Regexp
	compute func([]float64) float64
	name    string
}

// NewStatisticsAnswerer creates a new StatisticsAnswerer.
func NewStatisticsAnswerer() *StatisticsAnswerer {
	return &StatisticsAnswerer{
		info: AnswererInfo{
			ID:          "statistics",
			Name:        "Statistics Calculator",
			Description: "Calculate statistics on number lists",
			Keywords:    []string{"min", "max", "avg", "average", "sum", "prod", "product", "range", "mean"},
			Examples: []string{
				"min 5 3 9 1 7",
				"max 10 20 15",
				"avg 1 2 3 4 5",
				"sum 10 20 30",
				"prod 2 3 4",
				"range 5 10 15 20",
			},
		},
		patterns: []*statsPattern{
			{
				re:   regexp.MustCompile(`(?i)^min\s+(.+)$`),
				name: "min",
				compute: func(nums []float64) float64 {
					if len(nums) == 0 {
						return 0
					}
					m := nums[0]
					for _, n := range nums[1:] {
						if n < m {
							m = n
						}
					}
					return m
				},
			},
			{
				re:   regexp.MustCompile(`(?i)^max\s+(.+)$`),
				name: "max",
				compute: func(nums []float64) float64 {
					if len(nums) == 0 {
						return 0
					}
					m := nums[0]
					for _, n := range nums[1:] {
						if n > m {
							m = n
						}
					}
					return m
				},
			},
			{
				re:   regexp.MustCompile(`(?i)^(?:avg|average|mean)\s+(.+)$`),
				name: "avg",
				compute: func(nums []float64) float64 {
					if len(nums) == 0 {
						return 0
					}
					sum := 0.0
					for _, n := range nums {
						sum += n
					}
					return sum / float64(len(nums))
				},
			},
			{
				re:   regexp.MustCompile(`(?i)^sum\s+(.+)$`),
				name: "sum",
				compute: func(nums []float64) float64 {
					sum := 0.0
					for _, n := range nums {
						sum += n
					}
					return sum
				},
			},
			{
				re:   regexp.MustCompile(`(?i)^(?:prod|product)\s+(.+)$`),
				name: "prod",
				compute: func(nums []float64) float64 {
					if len(nums) == 0 {
						return 0
					}
					prod := 1.0
					for _, n := range nums {
						prod *= n
					}
					return prod
				},
			},
			{
				re:   regexp.MustCompile(`(?i)^range\s+(.+)$`),
				name: "range",
				compute: func(nums []float64) float64 {
					if len(nums) == 0 {
						return 0
					}
					min, max := nums[0], nums[0]
					for _, n := range nums[1:] {
						if n < min {
							min = n
						}
						if n > max {
							max = n
						}
					}
					return max - min
				},
			},
		},
	}
}

func (a *StatisticsAnswerer) Keywords() []string { return a.info.Keywords }
func (a *StatisticsAnswerer) Info() AnswererInfo { return a.info }

func (a *StatisticsAnswerer) Answer(ctx context.Context, query string) []engines.Answer {
	query = strings.TrimSpace(query)

	for _, pattern := range a.patterns {
		matches := pattern.re.FindStringSubmatch(query)
		if len(matches) >= 2 {
			numsStr := matches[1]
			nums := parseNumbers(numsStr)
			if len(nums) > 0 {
				result := pattern.compute(nums)
				return []engines.Answer{{
					Answer: fmt.Sprintf("%s(%s) = %s", pattern.name, numsStr, formatNumber(result)),
				}}
			}
		}
	}

	return nil
}

// parseNumbers extracts numbers from a string.
func parseNumbers(s string) []float64 {
	parts := strings.Fields(s)
	nums := make([]float64, 0, len(parts))

	for _, part := range parts {
		// Remove common separators
		part = strings.Trim(part, ",;")
		if part == "" {
			continue
		}

		var num float64
		n, err := fmt.Sscanf(part, "%f", &num)
		if err == nil && n == 1 {
			nums = append(nums, num)
		}
	}

	return nums
}

// formatNumber formats a number for display.
func formatNumber(n float64) string {
	// Check if it's an integer
	if n == float64(int64(n)) {
		return fmt.Sprintf("%d", int64(n))
	}
	// Format with up to 6 decimal places, removing trailing zeros
	s := fmt.Sprintf("%.6f", n)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

// ColorAnswerer provides color conversions.
type ColorAnswerer struct {
	info AnswererInfo
}

// NewColorAnswerer creates a new ColorAnswerer.
func NewColorAnswerer() *ColorAnswerer {
	return &ColorAnswerer{
		info: AnswererInfo{
			ID:          "color",
			Name:        "Color Converter",
			Description: "Convert between color formats (RGB, HEX)",
			Keywords:    []string{"rgb", "hex", "color", "#"},
			Examples: []string{
				"#ff5500",
				"rgb 255 85 0",
				"hex 255 128 64",
			},
		},
	}
}

func (a *ColorAnswerer) Keywords() []string { return a.info.Keywords }
func (a *ColorAnswerer) Info() AnswererInfo { return a.info }

func (a *ColorAnswerer) Answer(ctx context.Context, query string) []engines.Answer {
	query = strings.TrimSpace(query)

	// Match hex color
	hexPattern := regexp.MustCompile(`^#?([0-9a-fA-F]{6})$`)
	if matches := hexPattern.FindStringSubmatch(query); len(matches) >= 2 {
		hex := matches[1]
		r, _ := parseHexByte(hex[0:2])
		g, _ := parseHexByte(hex[2:4])
		b, _ := parseHexByte(hex[4:6])
		return []engines.Answer{{
			Answer: fmt.Sprintf("#%s = RGB(%d, %d, %d)", strings.ToUpper(hex), r, g, b),
		}}
	}

	// Match rgb to hex
	rgbPattern := regexp.MustCompile(`(?i)^(?:rgb|hex)\s+(\d+)\s+(\d+)\s+(\d+)$`)
	if matches := rgbPattern.FindStringSubmatch(query); len(matches) >= 4 {
		var r, g, b int
		fmt.Sscanf(matches[1], "%d", &r)
		fmt.Sscanf(matches[2], "%d", &g)
		fmt.Sscanf(matches[3], "%d", &b)
		if r >= 0 && r <= 255 && g >= 0 && g <= 255 && b >= 0 && b <= 255 {
			return []engines.Answer{{
				Answer: fmt.Sprintf("RGB(%d, %d, %d) = #%02X%02X%02X", r, g, b, r, g, b),
			}}
		}
	}

	return nil
}

func parseHexByte(s string) (int, error) {
	var result int
	for _, c := range strings.ToLower(s) {
		result *= 16
		if c >= '0' && c <= '9' {
			result += int(c - '0')
		} else if c >= 'a' && c <= 'f' {
			result += int(c-'a') + 10
		} else {
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
	}
	return result, nil
}
