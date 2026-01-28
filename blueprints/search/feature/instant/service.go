package instant

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// Service implements the instant answer API.
type Service struct{}

// NewService creates a new instant answer service.
func NewService() *Service {
	return &Service{}
}

// Calculate evaluates a mathematical expression.
func (s *Service) Calculate(expr string) (*store.InstantAnswer, error) {
	result, err := evaluateExpression(expr)
	if err != nil {
		return nil, err
	}

	return &store.InstantAnswer{
		Type:   "calculator",
		Query:  expr,
		Result: formatNumber(result),
		Data: store.CalculatorResult{
			Expression: expr,
			Result:     result,
			Formatted:  formatNumber(result),
		},
	}, nil
}

// ConvertUnit converts between units.
func (s *Service) ConvertUnit(value float64, from, to string) (*store.InstantAnswer, error) {
	result, category, err := convertUnit(value, from, to)
	if err != nil {
		return nil, err
	}

	return &store.InstantAnswer{
		Type:   "unit",
		Query:  fmt.Sprintf("%.6g %s to %s", value, from, to),
		Result: fmt.Sprintf("%s %s", formatNumber(result), to),
		Data: store.UnitConversionResult{
			FromValue: value,
			FromUnit:  from,
			ToValue:   result,
			ToUnit:    to,
			Category:  category,
		},
	}, nil
}

// ConvertCurrency converts between currencies.
func (s *Service) ConvertCurrency(amount float64, from, to string) (*store.InstantAnswer, error) {
	result, rate, err := convertCurrency(amount, strings.ToUpper(from), strings.ToUpper(to))
	if err != nil {
		return nil, err
	}

	return &store.InstantAnswer{
		Type:   "currency",
		Query:  fmt.Sprintf("%.2f %s to %s", amount, from, to),
		Result: fmt.Sprintf("%.2f %s", result, to),
		Data: store.CurrencyResult{
			FromAmount:   amount,
			FromCurrency: strings.ToUpper(from),
			ToAmount:     result,
			ToCurrency:   strings.ToUpper(to),
			Rate:         rate,
			UpdatedAt:    time.Now().Format(time.RFC3339),
		},
	}, nil
}

// GetWeather returns weather for a location.
func (s *Service) GetWeather(location string) *store.InstantAnswer {
	if location == "" {
		location = "New York"
	}

	weather := store.WeatherResult{
		Location:    location,
		Temperature: 22.5,
		Unit:        "C",
		Condition:   "Partly Cloudy",
		Humidity:    65,
		WindSpeed:   12.5,
		WindUnit:    "km/h",
		Icon:        "partly-cloudy",
	}

	return &store.InstantAnswer{
		Type:   "weather",
		Query:  fmt.Sprintf("weather %s", location),
		Result: fmt.Sprintf("%.0f%s, %s", weather.Temperature, weather.Unit, weather.Condition),
		Data:   weather,
	}
}

// Define returns dictionary definition for a word.
func (s *Service) Define(word string) (*store.InstantAnswer, error) {
	definition := getDefinition(word)
	if definition == nil {
		return nil, fmt.Errorf("definition not found")
	}

	return &store.InstantAnswer{
		Type:   "definition",
		Query:  fmt.Sprintf("define %s", word),
		Result: definition.Definitions[0],
		Data:   definition,
	}, nil
}

// GetTime returns current time for a location.
func (s *Service) GetTime(location string) *store.InstantAnswer {
	if location == "" {
		location = "UTC"
	}

	loc, err := time.LoadLocation(getTimezone(location))
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)

	return &store.InstantAnswer{
		Type:   "time",
		Query:  fmt.Sprintf("time in %s", location),
		Result: now.Format("3:04 PM"),
		Data: store.TimeResult{
			Location: location,
			Time:     now.Format("3:04:05 PM"),
			Date:     now.Format("Monday, January 2, 2006"),
			Timezone: loc.String(),
			Offset:   now.Format("-07:00"),
		},
	}
}

// Detect attempts to detect and compute an instant answer from a query.
func (s *Service) Detect(query string) *store.InstantAnswer {
	query = strings.TrimSpace(strings.ToLower(query))

	// Calculator detection
	if isCalculatorQuery(query) {
		result, err := evaluateExpression(query)
		if err == nil {
			return &store.InstantAnswer{
				Type:   "calculator",
				Query:  query,
				Result: formatNumber(result),
				Data: store.CalculatorResult{
					Expression: query,
					Result:     result,
					Formatted:  formatNumber(result),
				},
			}
		}
	}

	// Unit conversion detection
	if match := unitConversionRegex.FindStringSubmatch(query); match != nil {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			from := strings.ToLower(match[2])
			to := strings.ToLower(match[3])
			if result, category, err := convertUnit(val, from, to); err == nil {
				return &store.InstantAnswer{
					Type:   "unit",
					Query:  query,
					Result: fmt.Sprintf("%s %s", formatNumber(result), to),
					Data: store.UnitConversionResult{
						FromValue: val,
						FromUnit:  from,
						ToValue:   result,
						ToUnit:    to,
						Category:  category,
					},
				}
			}
		}
	}

	// Currency conversion detection
	if match := currencyRegex.FindStringSubmatch(query); match != nil {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			from := strings.ToUpper(match[2])
			to := strings.ToUpper(match[3])
			if result, rate, err := convertCurrency(val, from, to); err == nil {
				return &store.InstantAnswer{
					Type:   "currency",
					Query:  query,
					Result: fmt.Sprintf("%.2f %s", result, to),
					Data: store.CurrencyResult{
						FromAmount:   val,
						FromCurrency: from,
						ToAmount:     result,
						ToCurrency:   to,
						Rate:         rate,
						UpdatedAt:    time.Now().Format(time.RFC3339),
					},
				}
			}
		}
	}

	// Weather detection
	if strings.HasPrefix(query, "weather ") || query == "weather" {
		location := strings.TrimPrefix(query, "weather ")
		return s.GetWeather(location)
	}

	// Time detection
	if strings.HasPrefix(query, "time in ") || query == "time" {
		location := strings.TrimPrefix(query, "time in ")
		return s.GetTime(location)
	}

	// Definition detection
	if strings.HasPrefix(query, "define ") {
		word := strings.TrimPrefix(query, "define ")
		if answer, err := s.Define(word); err == nil {
			return answer
		}
	}

	return nil
}

// Helper functions

var (
	unitConversionRegex = regexp.MustCompile(`^([\d.]+)\s*(\w+)\s+(?:to|in|=)\s+(\w+)$`)
	currencyRegex       = regexp.MustCompile(`^([\d.]+)\s*(usd|eur|gbp|jpy|cny|aud|cad|chf|hkd|sgd|btc|eth)\s+(?:to|in|=)\s+(usd|eur|gbp|jpy|cny|aud|cad|chf|hkd|sgd|btc|eth)$`)
	calculatorRegex     = regexp.MustCompile(`^[\d\s+\-*/().^%]+$`)
)

func isCalculatorQuery(query string) bool {
	return calculatorRegex.MatchString(query)
}

func evaluateExpression(expr string) (float64, error) {
	expr = strings.ReplaceAll(expr, " ", "")
	expr = strings.ReplaceAll(expr, "^", "**")

	node, err := parser.ParseExpr(expr)
	if err != nil {
		return 0, fmt.Errorf("invalid expression")
	}

	return evalNode(node)
}

func evalNode(node ast.Expr) (float64, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		return strconv.ParseFloat(n.Value, 64)
	case *ast.ParenExpr:
		return evalNode(n.X)
	case *ast.BinaryExpr:
		left, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}
		right, err := evalNode(n.Y)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.ADD:
			return left + right, nil
		case token.SUB:
			return left - right, nil
		case token.MUL:
			return left * right, nil
		case token.QUO:
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		case token.REM:
			return math.Mod(left, right), nil
		}
	case *ast.UnaryExpr:
		val, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}
		if n.Op == token.SUB {
			return -val, nil
		}
		return val, nil
	}
	return 0, fmt.Errorf("unsupported expression")
}

func formatNumber(n float64) string {
	if n == float64(int64(n)) {
		return fmt.Sprintf("%.0f", n)
	}
	return fmt.Sprintf("%.6g", n)
}

// Unit conversions
var unitConversions = map[string]map[string]float64{
	"length": {
		"m": 1, "meter": 1, "meters": 1,
		"km": 1000, "kilometer": 1000, "kilometers": 1000,
		"cm": 0.01, "centimeter": 0.01, "centimeters": 0.01,
		"mm": 0.001, "millimeter": 0.001, "millimeters": 0.001,
		"mi": 1609.344, "mile": 1609.344, "miles": 1609.344,
		"ft": 0.3048, "foot": 0.3048, "feet": 0.3048,
		"in": 0.0254, "inch": 0.0254, "inches": 0.0254,
		"yd": 0.9144, "yard": 0.9144, "yards": 0.9144,
	},
	"weight": {
		"g": 1, "gram": 1, "grams": 1,
		"kg": 1000, "kilogram": 1000, "kilograms": 1000,
		"mg": 0.001, "milligram": 0.001, "milligrams": 0.001,
		"lb": 453.592, "pound": 453.592, "pounds": 453.592,
		"oz": 28.3495, "ounce": 28.3495, "ounces": 28.3495,
		"t": 1000000, "ton": 1000000, "tons": 1000000,
	},
	"temperature": {
		"c": 1, "celsius": 1,
		"f": 1, "fahrenheit": 1,
		"k": 1, "kelvin": 1,
	},
	"volume": {
		"l": 1, "liter": 1, "liters": 1,
		"ml": 0.001, "milliliter": 0.001, "milliliters": 0.001,
		"gal": 3.78541, "gallon": 3.78541, "gallons": 3.78541,
		"qt": 0.946353, "quart": 0.946353, "quarts": 0.946353,
		"pt": 0.473176, "pint": 0.473176, "pints": 0.473176,
		"cup": 0.236588, "cups": 0.236588,
	},
	"digital": {
		"b": 1, "byte": 1, "bytes": 1,
		"kb": 1024, "kilobyte": 1024, "kilobytes": 1024,
		"mb": 1048576, "megabyte": 1048576, "megabytes": 1048576,
		"gb": 1073741824, "gigabyte": 1073741824, "gigabytes": 1073741824,
		"tb": 1099511627776, "terabyte": 1099511627776, "terabytes": 1099511627776,
	},
}

func convertUnit(value float64, from, to string) (float64, string, error) {
	from = strings.ToLower(from)
	to = strings.ToLower(to)

	if isTemperature(from) && isTemperature(to) {
		result := convertTemperature(value, from, to)
		return result, "temperature", nil
	}

	for category, units := range unitConversions {
		fromFactor, fromOK := units[from]
		toFactor, toOK := units[to]
		if fromOK && toOK {
			result := value * fromFactor / toFactor
			return result, category, nil
		}
	}

	return 0, "", fmt.Errorf("unsupported conversion")
}

func isTemperature(unit string) bool {
	unit = strings.ToLower(unit)
	return unit == "c" || unit == "celsius" || unit == "f" || unit == "fahrenheit" || unit == "k" || unit == "kelvin"
}

func convertTemperature(value float64, from, to string) float64 {
	from = strings.ToLower(from)
	to = strings.ToLower(to)

	var celsius float64
	switch from {
	case "c", "celsius":
		celsius = value
	case "f", "fahrenheit":
		celsius = (value - 32) * 5 / 9
	case "k", "kelvin":
		celsius = value - 273.15
	}

	switch to {
	case "c", "celsius":
		return celsius
	case "f", "fahrenheit":
		return celsius*9/5 + 32
	case "k", "kelvin":
		return celsius + 273.15
	}

	return value
}

var currencyRates = map[string]float64{
	"USD": 1.0,
	"EUR": 0.92,
	"GBP": 0.79,
	"JPY": 149.50,
	"CNY": 7.24,
	"AUD": 1.53,
	"CAD": 1.36,
	"CHF": 0.88,
	"HKD": 7.82,
	"SGD": 1.34,
	"BTC": 0.000023,
	"ETH": 0.00035,
}

func convertCurrency(amount float64, from, to string) (float64, float64, error) {
	fromRate, fromOK := currencyRates[from]
	toRate, toOK := currencyRates[to]
	if !fromOK || !toOK {
		return 0, 0, fmt.Errorf("unsupported currency")
	}

	usdAmount := amount / fromRate
	result := usdAmount * toRate
	rate := toRate / fromRate

	return result, rate, nil
}

var definitions = map[string]*store.DefinitionResult{
	"programming": {
		Word:         "programming",
		Phonetic:     "/ˈprəʊɡræmɪŋ/",
		PartOfSpeech: "noun",
		Definitions:  []string{"The process of writing computer programs.", "The action or process of scheduling something."},
		Synonyms:     []string{"coding", "software development"},
		Examples:     []string{"She studied programming in college.", "The programming of the event took weeks."},
	},
	"algorithm": {
		Word:         "algorithm",
		Phonetic:     "/ˈælɡərɪðəm/",
		PartOfSpeech: "noun",
		Definitions:  []string{"A process or set of rules to be followed in calculations or problem-solving operations."},
		Synonyms:     []string{"procedure", "method", "formula"},
		Examples:     []string{"The search algorithm finds the shortest path."},
	},
	"search": {
		Word:         "search",
		Phonetic:     "/sɜːtʃ/",
		PartOfSpeech: "verb",
		Definitions:  []string{"Try to find something by looking or otherwise seeking carefully and thoroughly.", "An act of searching for something."},
		Synonyms:     []string{"look for", "hunt", "seek"},
		Examples:     []string{"I searched the entire house.", "The search for meaning continues."},
	},
	"database": {
		Word:         "database",
		Phonetic:     "/ˈdeɪtəbeɪs/",
		PartOfSpeech: "noun",
		Definitions:  []string{"A structured set of data held in a computer, especially one that is accessible in various ways."},
		Synonyms:     []string{"data bank", "data store", "repository"},
		Examples:     []string{"The customer database contains millions of records."},
	},
	"api": {
		Word:         "API",
		Phonetic:     "/ˌeɪpiːˈaɪ/",
		PartOfSpeech: "noun",
		Definitions:  []string{"Application Programming Interface - a set of functions and procedures allowing the creation of applications that access features or data of an operating system, application, or other service."},
		Synonyms:     []string{"interface", "connector"},
		Examples:     []string{"The REST API allows external applications to access our data."},
	},
	"framework": {
		Word:         "framework",
		Phonetic:     "/ˈfreɪmwɜːk/",
		PartOfSpeech: "noun",
		Definitions:  []string{"A basic structure underlying a system, concept, or text.", "A software framework provides a foundation on which software developers can build programs."},
		Synonyms:     []string{"structure", "foundation", "scaffold"},
		Examples:     []string{"React is a popular JavaScript framework."},
	},
}

func getDefinition(word string) *store.DefinitionResult {
	word = strings.ToLower(strings.TrimSpace(word))
	return definitions[word]
}

var timezones = map[string]string{
	"new york":    "America/New_York",
	"los angeles": "America/Los_Angeles",
	"chicago":     "America/Chicago",
	"london":      "Europe/London",
	"paris":       "Europe/Paris",
	"berlin":      "Europe/Berlin",
	"tokyo":       "Asia/Tokyo",
	"sydney":      "Australia/Sydney",
	"beijing":     "Asia/Shanghai",
	"moscow":      "Europe/Moscow",
	"dubai":       "Asia/Dubai",
	"singapore":   "Asia/Singapore",
	"hong kong":   "Asia/Hong_Kong",
	"mumbai":      "Asia/Kolkata",
	"seoul":       "Asia/Seoul",
}

func getTimezone(location string) string {
	location = strings.ToLower(strings.TrimSpace(location))
	if tz, ok := timezones[location]; ok {
		return tz
	}
	return "UTC"
}
