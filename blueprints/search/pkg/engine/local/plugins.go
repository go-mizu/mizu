package local

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local/engines"
)

// Plugin defines the plugin interface.
type Plugin interface {
	// ID returns unique plugin identifier.
	ID() string

	// Info returns plugin metadata.
	Info() PluginInfo

	// Active returns true if plugin is active by default.
	Active() bool
}

// PreSearchPlugin can modify search before execution.
type PreSearchPlugin interface {
	Plugin

	// PreSearch is called before search execution.
	// Return false to stop the search.
	PreSearch(ctx context.Context, search *Search) bool
}

// PostSearchPlugin can add results after search.
type PostSearchPlugin interface {
	Plugin

	// PostSearch is called after search execution.
	PostSearch(ctx context.Context, search *Search)
}

// OnResultPlugin can filter/modify individual results.
type OnResultPlugin interface {
	Plugin

	// OnResult is called for each result.
	// Return false to remove the result.
	OnResult(ctx context.Context, search *Search, result *engines.Result) bool
}

// PluginInfo contains plugin metadata.
type PluginInfo struct {
	ID          string
	Name        string
	Description string
	Keywords    []string
}

// PluginStorage manages plugins.
type PluginStorage struct {
	mu       sync.RWMutex
	plugins  map[string]Plugin
	enabled  map[string]bool
	keywords map[string][]string
}

// NewPluginStorage creates a new PluginStorage.
func NewPluginStorage() *PluginStorage {
	return &PluginStorage{
		plugins:  make(map[string]Plugin),
		enabled:  make(map[string]bool),
		keywords: make(map[string][]string),
	}
}

// Register registers a plugin.
func (ps *PluginStorage) Register(plugin Plugin) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	id := plugin.ID()
	ps.plugins[id] = plugin
	ps.enabled[id] = plugin.Active()

	info := plugin.Info()
	if len(info.Keywords) > 0 {
		ps.keywords[id] = info.Keywords
	}
}

// Enable enables a plugin.
func (ps *PluginStorage) Enable(id string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.enabled[id] = true
}

// Disable disables a plugin.
func (ps *PluginStorage) Disable(id string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.enabled[id] = false
}

// IsEnabled returns whether a plugin is enabled.
func (ps *PluginStorage) IsEnabled(id string) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.enabled[id]
}

// PreSearch runs all pre-search plugins.
func (ps *PluginStorage) PreSearch(ctx context.Context, search *Search) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for id, plugin := range ps.plugins {
		if !ps.enabled[id] {
			continue
		}
		if p, ok := plugin.(PreSearchPlugin); ok {
			if !p.PreSearch(ctx, search) {
				return false
			}
		}
	}
	return true
}

// PostSearch runs all post-search plugins.
func (ps *PluginStorage) PostSearch(ctx context.Context, search *Search) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for id, plugin := range ps.plugins {
		if !ps.enabled[id] {
			continue
		}
		if p, ok := plugin.(PostSearchPlugin); ok {
			p.PostSearch(ctx, search)
		}
	}
}

// OnResult runs all on-result plugins.
func (ps *PluginStorage) OnResult(ctx context.Context, search *Search, result *engines.Result) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for id, plugin := range ps.plugins {
		if !ps.enabled[id] {
			continue
		}
		if p, ok := plugin.(OnResultPlugin); ok {
			if !p.OnResult(ctx, search, result) {
				return false
			}
		}
	}
	return true
}

// TrackerURLRemoverPlugin removes tracking parameters from URLs.
type TrackerURLRemoverPlugin struct {
	info     PluginInfo
	patterns []*regexp.Regexp
}

// NewTrackerURLRemoverPlugin creates a new TrackerURLRemoverPlugin.
func NewTrackerURLRemoverPlugin() *TrackerURLRemoverPlugin {
	trackingParams := []string{
		`^utm_\w+$`,
		`^ga_\w+$`,
		`^fbclid$`,
		`^gclid$`,
		`^msclkid$`,
		`^mc_eid$`,
		`^dclid$`,
		`^yclid$`,
		`^_ga$`,
		`^_gl$`,
		`^__hsfp$`,
		`^__hssc$`,
		`^__hstc$`,
		`^_hsenc$`,
		`^hsCtaTracking$`,
		`^ref$`,
		`^ref_src$`,
		`^ref_url$`,
		`^s_kwcid$`,
	}

	patterns := make([]*regexp.Regexp, len(trackingParams))
	for i, p := range trackingParams {
		patterns[i] = regexp.MustCompile(p)
	}

	return &TrackerURLRemoverPlugin{
		info: PluginInfo{
			ID:          "tracker_url_remover",
			Name:        "Tracker URL Remover",
			Description: "Remove tracking parameters from result URLs",
		},
		patterns: patterns,
	}
}

func (p *TrackerURLRemoverPlugin) ID() string        { return p.info.ID }
func (p *TrackerURLRemoverPlugin) Info() PluginInfo  { return p.info }
func (p *TrackerURLRemoverPlugin) Active() bool      { return true }

func (p *TrackerURLRemoverPlugin) OnResult(ctx context.Context, search *Search, result *engines.Result) bool {
	if result.URL == "" {
		return true
	}

	u, err := url.Parse(result.URL)
	if err != nil {
		return true
	}

	query := u.Query()
	changed := false

	for key := range query {
		for _, pattern := range p.patterns {
			if pattern.MatchString(key) {
				query.Del(key)
				changed = true
				break
			}
		}
	}

	if changed {
		u.RawQuery = query.Encode()
		result.URL = u.String()
		result.ParsedURL = u
	}

	return true
}

// HostnameBlockerPlugin blocks results from specified hostnames.
type HostnameBlockerPlugin struct {
	info      PluginInfo
	hostnames map[string]struct{}
}

// NewHostnameBlockerPlugin creates a new HostnameBlockerPlugin.
func NewHostnameBlockerPlugin(hostnames []string) *HostnameBlockerPlugin {
	hostnameSet := make(map[string]struct{})
	for _, h := range hostnames {
		hostnameSet[strings.ToLower(h)] = struct{}{}
	}

	return &HostnameBlockerPlugin{
		info: PluginInfo{
			ID:          "hostname_blocker",
			Name:        "Hostname Blocker",
			Description: "Block results from specified hostnames",
		},
		hostnames: hostnameSet,
	}
}

func (p *HostnameBlockerPlugin) ID() string        { return p.info.ID }
func (p *HostnameBlockerPlugin) Info() PluginInfo  { return p.info }
func (p *HostnameBlockerPlugin) Active() bool      { return false }

func (p *HostnameBlockerPlugin) OnResult(ctx context.Context, search *Search, result *engines.Result) bool {
	if result.ParsedURL == nil {
		return true
	}

	host := strings.ToLower(result.ParsedURL.Hostname())

	// Check exact match
	if _, blocked := p.hostnames[host]; blocked {
		return false
	}

	// Check with www prefix removed
	if strings.HasPrefix(host, "www.") {
		if _, blocked := p.hostnames[host[4:]]; blocked {
			return false
		}
	}

	return true
}

// SetHostnames updates the blocked hostnames.
func (p *HostnameBlockerPlugin) SetHostnames(hostnames []string) {
	hostnameSet := make(map[string]struct{})
	for _, h := range hostnames {
		hostnameSet[strings.ToLower(h)] = struct{}{}
	}
	p.hostnames = hostnameSet
}

// HashPlugin provides hash calculation answers.
type HashPlugin struct {
	info     PluginInfo
	patterns []*hashPattern
}

type hashPattern struct {
	re       *regexp.Regexp
	hashFunc func(string) string
	name     string
}

// NewHashPlugin creates a new HashPlugin.
func NewHashPlugin() *HashPlugin {
	return &HashPlugin{
		info: PluginInfo{
			ID:          "hash_plugin",
			Name:        "Hash Calculator",
			Description: "Calculate hashes of strings",
			Keywords:    []string{"md5", "sha1", "sha256", "sha512"},
		},
		patterns: []*hashPattern{
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

func (p *HashPlugin) ID() string        { return p.info.ID }
func (p *HashPlugin) Info() PluginInfo  { return p.info }
func (p *HashPlugin) Active() bool      { return true }

func (p *HashPlugin) PreSearch(ctx context.Context, search *Search) bool {
	query := strings.TrimSpace(search.Query)

	for _, pattern := range p.patterns {
		matches := pattern.re.FindStringSubmatch(query)
		if len(matches) >= 2 {
			input := matches[1]
			hash := pattern.hashFunc(input)
			search.Container.answers = append(search.Container.answers, engines.Answer{
				Answer: pattern.name + ": " + hash,
			})
		}
	}

	return true
}

// HostnameReplacerPlugin replaces hostnames in results.
type HostnameReplacerPlugin struct {
	info         PluginInfo
	replacements map[string]string
}

// NewHostnameReplacerPlugin creates a new HostnameReplacerPlugin.
func NewHostnameReplacerPlugin(replacements map[string]string) *HostnameReplacerPlugin {
	return &HostnameReplacerPlugin{
		info: PluginInfo{
			ID:          "hostname_replacer",
			Name:        "Hostname Replacer",
			Description: "Replace hostnames in result URLs",
		},
		replacements: replacements,
	}
}

func (p *HostnameReplacerPlugin) ID() string        { return p.info.ID }
func (p *HostnameReplacerPlugin) Info() PluginInfo  { return p.info }
func (p *HostnameReplacerPlugin) Active() bool      { return false }

func (p *HostnameReplacerPlugin) OnResult(ctx context.Context, search *Search, result *engines.Result) bool {
	if result.ParsedURL == nil {
		return true
	}

	host := result.ParsedURL.Hostname()
	if replacement, ok := p.replacements[host]; ok {
		result.ParsedURL.Host = replacement
		result.URL = result.ParsedURL.String()
	}

	return true
}

// SelfInfoPlugin shows user's IP and user agent.
type SelfInfoPlugin struct {
	info PluginInfo
}

// NewSelfInfoPlugin creates a new SelfInfoPlugin.
func NewSelfInfoPlugin() *SelfInfoPlugin {
	return &SelfInfoPlugin{
		info: PluginInfo{
			ID:          "self_info",
			Name:        "Self Information",
			Description: "Displays your IP if the query is 'ip' and your user agent if the query is 'user-agent'",
			Keywords:    []string{"ip", "user-agent", "user agent", "my ip", "what is my ip"},
		},
	}
}

func (p *SelfInfoPlugin) ID() string       { return p.info.ID }
func (p *SelfInfoPlugin) Info() PluginInfo { return p.info }
func (p *SelfInfoPlugin) Active() bool     { return false }

func (p *SelfInfoPlugin) PostSearch(ctx context.Context, search *Search) {
	query := strings.ToLower(strings.TrimSpace(search.Query))

	switch query {
	case "ip", "my ip", "what is my ip", "myip":
		// In a real implementation, this would get the client's IP
		// For now, we indicate this needs to be set by the caller
		search.Container.answers = append(search.Container.answers, engines.Answer{
			Answer: "Your IP address (set by server)",
		})
	case "user-agent", "user agent", "useragent", "my user agent":
		search.Container.answers = append(search.Container.answers, engines.Answer{
			Answer: "Your User-Agent (set by server)",
		})
	}
}

// CalculatorPlugin evaluates mathematical expressions.
type CalculatorPlugin struct {
	info    PluginInfo
	pattern *regexp.Regexp
}

// NewCalculatorPlugin creates a new CalculatorPlugin.
func NewCalculatorPlugin() *CalculatorPlugin {
	return &CalculatorPlugin{
		info: PluginInfo{
			ID:          "calculator",
			Name:        "Calculator",
			Description: "Parses and solves mathematical expressions",
			Keywords:    []string{"="},
		},
		// Match simple math expressions
		pattern: regexp.MustCompile(`^[\d\s\+\-\*\/\(\)\.\^%]+$`),
	}
}

func (p *CalculatorPlugin) ID() string       { return p.info.ID }
func (p *CalculatorPlugin) Info() PluginInfo { return p.info }
func (p *CalculatorPlugin) Active() bool     { return false }

func (p *CalculatorPlugin) PostSearch(ctx context.Context, search *Search) {
	query := strings.TrimSpace(search.Query)

	// Check if it looks like a math expression
	if !p.pattern.MatchString(query) {
		return
	}

	// Simple evaluation for basic arithmetic
	result, err := evaluateExpression(query)
	if err == nil {
		search.Container.answers = append(search.Container.answers, engines.Answer{
			Answer: query + " = " + result,
		})
	}
}

// evaluateExpression evaluates a simple math expression.
// This is a basic implementation - a full calculator would use a proper parser.
func evaluateExpression(expr string) (string, error) {
	// Remove spaces
	expr = strings.ReplaceAll(expr, " ", "")

	// For safety, only allow basic math characters
	for _, c := range expr {
		if !strings.ContainsRune("0123456789.+-*/()^%", c) {
			return "", nil
		}
	}

	// This is a placeholder - a real implementation would use a proper expression parser
	// For now, we'll just indicate the expression was recognized
	return "(calculator result)", nil
}

// UnitConverterPlugin converts between units.
type UnitConverterPlugin struct {
	info       PluginInfo
	pattern    *regexp.Regexp
	conversions map[string]map[string]float64
}

// NewUnitConverterPlugin creates a new UnitConverterPlugin.
func NewUnitConverterPlugin() *UnitConverterPlugin {
	return &UnitConverterPlugin{
		info: PluginInfo{
			ID:          "unit_converter",
			Name:        "Unit Converter",
			Description: "Convert between units",
			Keywords:    []string{"in", "to", "convert"},
		},
		pattern: regexp.MustCompile(`(?i)^([\d.]+)\s*(\w+)\s+(?:in|to|as)\s+(\w+)$`),
		conversions: map[string]map[string]float64{
			// Length
			"m":  {"km": 0.001, "cm": 100, "mm": 1000, "mi": 0.000621371, "ft": 3.28084, "in": 39.3701, "yd": 1.09361},
			"km": {"m": 1000, "mi": 0.621371, "ft": 3280.84},
			"mi": {"km": 1.60934, "m": 1609.34, "ft": 5280},
			"ft": {"m": 0.3048, "in": 12, "cm": 30.48, "yd": 0.333333},
			"in": {"cm": 2.54, "mm": 25.4, "ft": 0.0833333},
			"cm": {"m": 0.01, "in": 0.393701, "mm": 10},
			"mm": {"cm": 0.1, "m": 0.001, "in": 0.0393701},
			"yd": {"m": 0.9144, "ft": 3},
			// Weight
			"kg":  {"g": 1000, "lb": 2.20462, "oz": 35.274},
			"g":   {"kg": 0.001, "mg": 1000, "oz": 0.035274},
			"lb":  {"kg": 0.453592, "oz": 16, "g": 453.592},
			"oz":  {"g": 28.3495, "lb": 0.0625},
			"mg":  {"g": 0.001},
			// Temperature (handled specially)
			// Volume
			"l":   {"ml": 1000, "gal": 0.264172, "qt": 1.05669},
			"ml":  {"l": 0.001},
			"gal": {"l": 3.78541, "qt": 4},
			"qt":  {"l": 0.946353, "gal": 0.25},
			// Time
			"s":   {"ms": 1000, "min": 0.0166667, "h": 0.000277778},
			"min": {"s": 60, "h": 0.0166667},
			"h":   {"min": 60, "s": 3600, "d": 0.0416667},
			"d":   {"h": 24, "min": 1440, "s": 86400},
			"ms":  {"s": 0.001},
		},
	}
}

func (p *UnitConverterPlugin) ID() string       { return p.info.ID }
func (p *UnitConverterPlugin) Info() PluginInfo { return p.info }
func (p *UnitConverterPlugin) Active() bool     { return false }

func (p *UnitConverterPlugin) PostSearch(ctx context.Context, search *Search) {
	matches := p.pattern.FindStringSubmatch(search.Query)
	if len(matches) != 4 {
		return
	}

	valueStr := matches[1]
	fromUnit := strings.ToLower(matches[2])
	toUnit := strings.ToLower(matches[3])

	var value float64
	if _, err := parseFloat(valueStr, &value); err != nil {
		return
	}

	// Handle temperature conversions specially
	if result, ok := p.convertTemperature(value, fromUnit, toUnit); ok {
		search.Container.answers = append(search.Container.answers, engines.Answer{
			Answer: result,
		})
		return
	}

	// Standard unit conversion
	if conversions, ok := p.conversions[fromUnit]; ok {
		if factor, ok := conversions[toUnit]; ok {
			result := value * factor
			search.Container.answers = append(search.Container.answers, engines.Answer{
				Answer: formatFloat(value) + " " + fromUnit + " = " + formatFloat(result) + " " + toUnit,
			})
		}
	}
}

func (p *UnitConverterPlugin) convertTemperature(value float64, from, to string) (string, bool) {
	var celsius float64

	// Convert to Celsius first
	switch from {
	case "c", "celsius":
		celsius = value
	case "f", "fahrenheit":
		celsius = (value - 32) * 5 / 9
	case "k", "kelvin":
		celsius = value - 273.15
	default:
		return "", false
	}

	// Convert from Celsius to target
	var result float64
	var toName string
	switch to {
	case "c", "celsius":
		result = celsius
		toName = "°C"
	case "f", "fahrenheit":
		result = celsius*9/5 + 32
		toName = "°F"
	case "k", "kelvin":
		result = celsius + 273.15
		toName = "K"
	default:
		return "", false
	}

	return formatFloat(value) + " → " + formatFloat(result) + " " + toName, true
}

func parseFloat(s string, v *float64) (bool, error) {
	var err error
	for _, c := range s {
		if c == '.' || (c >= '0' && c <= '9') || c == '-' {
			continue
		}
		return false, nil
	}
	_, err = stringToFloat(s, v)
	return err == nil, err
}

func stringToFloat(s string, v *float64) (bool, error) {
	// Simple string to float conversion
	var result float64
	var decimal float64 = 1
	var negative bool
	var afterDecimal bool

	for i, c := range s {
		if c == '-' && i == 0 {
			negative = true
			continue
		}
		if c == '.' {
			afterDecimal = true
			continue
		}
		if c >= '0' && c <= '9' {
			digit := float64(c - '0')
			if afterDecimal {
				decimal *= 10
				result += digit / decimal
			} else {
				result = result*10 + digit
			}
		}
	}

	if negative {
		result = -result
	}
	*v = result
	return true, nil
}

func formatFloat(v float64) string {
	// Format float with reasonable precision
	if v == float64(int64(v)) {
		return strings.TrimRight(strings.TrimRight(
			strings.Replace(string(rune(int64(v)+'0')), ".", "", 1), "0"), ".")
	}
	// Simple formatting
	s := ""
	if v < 0 {
		s = "-"
		v = -v
	}
	intPart := int64(v)
	fracPart := v - float64(intPart)

	// Format integer part
	if intPart == 0 {
		s += "0"
	} else {
		digits := ""
		for intPart > 0 {
			digits = string(rune('0'+intPart%10)) + digits
			intPart /= 10
		}
		s += digits
	}

	// Format fractional part (up to 6 decimal places)
	if fracPart > 0.000001 {
		s += "."
		for i := 0; i < 6 && fracPart > 0.000001; i++ {
			fracPart *= 10
			digit := int(fracPart)
			s += string(rune('0' + digit))
			fracPart -= float64(digit)
		}
		// Trim trailing zeros
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}

	return s
}

// OpenAccessDOIRewritePlugin rewrites DOI URLs to open access versions.
type OpenAccessDOIRewritePlugin struct {
	info      PluginInfo
	doiPattern *regexp.Regexp
}

// NewOpenAccessDOIRewritePlugin creates a new OpenAccessDOIRewritePlugin.
func NewOpenAccessDOIRewritePlugin() *OpenAccessDOIRewritePlugin {
	return &OpenAccessDOIRewritePlugin{
		info: PluginInfo{
			ID:          "oa_doi_rewrite",
			Name:        "Open Access DOI Rewrite",
			Description: "Avoid paywalls by redirecting to open-access versions of publications",
		},
		doiPattern: regexp.MustCompile(`10\.\d{4,}/[^\s]+`),
	}
}

func (p *OpenAccessDOIRewritePlugin) ID() string       { return p.info.ID }
func (p *OpenAccessDOIRewritePlugin) Info() PluginInfo { return p.info }
func (p *OpenAccessDOIRewritePlugin) Active() bool     { return false }

func (p *OpenAccessDOIRewritePlugin) OnResult(ctx context.Context, search *Search, result *engines.Result) bool {
	if result.URL == "" {
		return true
	}

	// Check if URL contains a DOI
	doi := p.doiPattern.FindString(result.URL)
	if doi == "" {
		// Also check in the DOI field if available
		if result.DOI != "" {
			doi = result.DOI
		}
	}

	if doi != "" {
		// Rewrite to use doi.org which often provides open access links
		result.URL = "https://doi.org/" + doi
		result.ParsedURL, _ = url.Parse(result.URL)
	}

	return true
}

// TimezonePlugin displays time in different timezones.
type TimezonePlugin struct {
	info PluginInfo
}

// NewTimezonePlugin creates a new TimezonePlugin.
func NewTimezonePlugin() *TimezonePlugin {
	return &TimezonePlugin{
		info: PluginInfo{
			ID:          "time_zone",
			Name:        "Timezones Plugin",
			Description: "Display the current time on different time zones",
			Keywords:    []string{"time", "timezone", "now", "clock", "timezones"},
		},
	}
}

func (p *TimezonePlugin) ID() string       { return p.info.ID }
func (p *TimezonePlugin) Info() PluginInfo { return p.info }
func (p *TimezonePlugin) Active() bool     { return false }

func (p *TimezonePlugin) PostSearch(ctx context.Context, search *Search) {
	query := strings.ToLower(strings.TrimSpace(search.Query))

	// Check if query starts with time-related keywords
	timeKeywords := []string{"time ", "time in ", "clock ", "now ", "timezone "}
	var location string
	for _, kw := range timeKeywords {
		if strings.HasPrefix(query, kw) {
			location = strings.TrimPrefix(query, kw)
			break
		}
	}

	if location == "" {
		return
	}

	// Map common location names to timezone identifiers
	timezones := map[string]string{
		"new york":     "America/New_York",
		"los angeles":  "America/Los_Angeles",
		"london":       "Europe/London",
		"paris":        "Europe/Paris",
		"berlin":       "Europe/Berlin",
		"tokyo":        "Asia/Tokyo",
		"sydney":       "Australia/Sydney",
		"beijing":      "Asia/Shanghai",
		"moscow":       "Europe/Moscow",
		"dubai":        "Asia/Dubai",
		"singapore":    "Asia/Singapore",
		"hong kong":    "Asia/Hong_Kong",
		"mumbai":       "Asia/Kolkata",
		"chicago":      "America/Chicago",
		"toronto":      "America/Toronto",
		"vancouver":    "America/Vancouver",
		"utc":          "UTC",
		"gmt":          "UTC",
	}

	tz, ok := timezones[strings.ToLower(location)]
	if !ok {
		// Try to use location as-is (might be a valid timezone)
		tz = location
	}

	// Note: In a real implementation, you would use time.LoadLocation(tz)
	// and format the current time. For this implementation, we indicate
	// what would be displayed.
	search.Container.answers = append(search.Container.answers, engines.Answer{
		Answer: "Current time in " + location + " (timezone: " + tz + ")",
	})
}
