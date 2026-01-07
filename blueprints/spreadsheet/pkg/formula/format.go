package formula

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FormatValue formats a value according to a number format string.
func FormatValue(value interface{}, format string) string {
	if value == nil {
		return ""
	}

	if format == "" {
		return fmt.Sprintf("%v", value)
	}

	// Handle string values
	if s, ok := value.(string); ok {
		if format == "@" {
			return s
		}
		// Try to parse as number for formatting
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			value = f
		} else {
			return s
		}
	}

	// Handle boolean
	if b, ok := value.(bool); ok {
		if b {
			return "TRUE"
		}
		return "FALSE"
	}

	// Get numeric value
	num, ok := toNumber(value)
	if !ok {
		return fmt.Sprintf("%v", value)
	}

	// Handle special formats
	formatLower := strings.ToLower(format)

	// Percentage format
	if strings.Contains(format, "%") {
		return formatPercentage(num, format)
	}

	// Currency format
	if strings.HasPrefix(format, "$") || strings.Contains(format, "Currency") {
		return formatCurrency(num, format)
	}

	// Scientific notation
	if strings.Contains(formatLower, "e+") || strings.Contains(formatLower, "e-") {
		return formatScientific(num, format)
	}

	// Date formats
	if isDateFormat(format) {
		return formatDate(num, format)
	}

	// Time formats
	if isTimeFormat(format) {
		return formatTime(num, format)
	}

	// General number format
	return formatNumber(num, format)
}

func formatPercentage(num float64, format string) string {
	// Count decimal places after %
	decimals := 0
	if idx := strings.Index(format, "%"); idx > 0 {
		beforePct := format[:idx]
		if dotIdx := strings.LastIndex(beforePct, "."); dotIdx >= 0 {
			decimals = len(beforePct) - dotIdx - 1
		}
	}

	return fmt.Sprintf("%.*f%%", decimals, num*100)
}

func formatCurrency(num float64, format string) string {
	decimals := 2
	if strings.Contains(format, ".00") {
		decimals = 2
	} else if strings.Contains(format, ".0") {
		decimals = 1
	}

	negative := num < 0
	if negative {
		num = -num
	}

	// Round to decimals
	mult := math.Pow(10, float64(decimals))
	rounded := math.Round(num*mult) / mult

	// Format with thousands separator
	intPart := int64(rounded)
	fracPart := rounded - float64(intPart)

	intStr := strconv.FormatInt(intPart, 10)
	var formatted strings.Builder
	for i, ch := range intStr {
		if i > 0 && (len(intStr)-i)%3 == 0 {
			formatted.WriteByte(',')
		}
		formatted.WriteRune(ch)
	}

	result := "$" + formatted.String()
	if decimals > 0 {
		fracStr := fmt.Sprintf("%.*f", decimals, fracPart)
		result += fracStr[1:]
	}

	if negative {
		result = "(" + result + ")"
	}

	return result
}

func formatScientific(num float64, format string) string {
	// Count decimal places
	decimals := 2
	if dotIdx := strings.Index(format, "."); dotIdx >= 0 {
		decimals = 0
		for i := dotIdx + 1; i < len(format); i++ {
			if format[i] == '0' || format[i] == '#' {
				decimals++
			} else {
				break
			}
		}
	}

	return fmt.Sprintf("%.*E", decimals, num)
}

func isDateFormat(format string) bool {
	lower := strings.ToLower(format)
	return strings.Contains(lower, "yyyy") ||
		strings.Contains(lower, "yy") ||
		strings.Contains(lower, "mm") ||
		strings.Contains(lower, "dd") ||
		strings.Contains(lower, "d-mmm") ||
		strings.Contains(lower, "mmm-yy")
}

func isTimeFormat(format string) bool {
	lower := strings.ToLower(format)
	return strings.Contains(lower, "hh") ||
		strings.Contains(lower, "ss") ||
		strings.Contains(lower, "am/pm") ||
		(strings.Contains(lower, ":") && !isDateFormat(format))
}

func formatDate(serial float64, format string) string {
	// Convert Excel serial date to time
	t := time.Unix(int64((serial-25569)*86400), 0).UTC()

	// Replace format tokens
	result := format

	// Year
	result = strings.ReplaceAll(result, "yyyy", strconv.Itoa(t.Year()))
	result = strings.ReplaceAll(result, "YYYY", strconv.Itoa(t.Year()))
	result = strings.ReplaceAll(result, "yy", fmt.Sprintf("%02d", t.Year()%100))
	result = strings.ReplaceAll(result, "YY", fmt.Sprintf("%02d", t.Year()%100))

	// Month (before day to avoid conflicts)
	monthNames := []string{"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}
	monthShort := []string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun",
		"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

	result = strings.ReplaceAll(result, "mmmm", monthNames[t.Month()])
	result = strings.ReplaceAll(result, "MMMM", monthNames[t.Month()])
	result = strings.ReplaceAll(result, "mmm", monthShort[t.Month()])
	result = strings.ReplaceAll(result, "MMM", monthShort[t.Month()])
	result = strings.ReplaceAll(result, "mm", fmt.Sprintf("%02d", t.Month()))
	result = strings.ReplaceAll(result, "MM", fmt.Sprintf("%02d", t.Month()))
	result = strings.ReplaceAll(result, "m", strconv.Itoa(int(t.Month())))
	result = strings.ReplaceAll(result, "M", strconv.Itoa(int(t.Month())))

	// Day
	dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	dayShort := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

	result = strings.ReplaceAll(result, "dddd", dayNames[t.Weekday()])
	result = strings.ReplaceAll(result, "DDDD", dayNames[t.Weekday()])
	result = strings.ReplaceAll(result, "ddd", dayShort[t.Weekday()])
	result = strings.ReplaceAll(result, "DDD", dayShort[t.Weekday()])
	result = strings.ReplaceAll(result, "dd", fmt.Sprintf("%02d", t.Day()))
	result = strings.ReplaceAll(result, "DD", fmt.Sprintf("%02d", t.Day()))
	result = strings.ReplaceAll(result, "d", strconv.Itoa(t.Day()))
	result = strings.ReplaceAll(result, "D", strconv.Itoa(t.Day()))

	return result
}

func formatTime(serial float64, format string) string {
	// Get time portion
	fraction := serial - math.Floor(serial)
	totalSeconds := int(fraction * 86400)
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	result := format

	// AM/PM handling
	isPM := hours >= 12
	ampm := "AM"
	if isPM {
		ampm = "PM"
	}

	hour12 := hours % 12
	if hour12 == 0 {
		hour12 = 12
	}

	// Replace tokens (case insensitive)
	re := regexp.MustCompile(`(?i)am/pm`)
	result = re.ReplaceAllString(result, ampm)

	re = regexp.MustCompile(`(?i)hh`)
	if strings.Contains(strings.ToLower(format), "am/pm") {
		result = re.ReplaceAllString(result, fmt.Sprintf("%02d", hour12))
	} else {
		result = re.ReplaceAllString(result, fmt.Sprintf("%02d", hours))
	}

	re = regexp.MustCompile(`(?i)h`)
	if strings.Contains(strings.ToLower(format), "am/pm") {
		result = re.ReplaceAllString(result, strconv.Itoa(hour12))
	} else {
		result = re.ReplaceAllString(result, strconv.Itoa(hours))
	}

	re = regexp.MustCompile(`(?i)mm`)
	result = re.ReplaceAllString(result, fmt.Sprintf("%02d", minutes))

	re = regexp.MustCompile(`(?i)ss`)
	result = re.ReplaceAllString(result, fmt.Sprintf("%02d", seconds))

	return result
}

func formatNumber(num float64, format string) string {
	// Parse format string to determine:
	// - Number of decimal places
	// - Use of thousands separator
	// - Padding

	hasComma := strings.Contains(format, ",")
	decimals := 0

	if dotIdx := strings.Index(format, "."); dotIdx >= 0 {
		// Count 0s or #s after decimal point
		for i := dotIdx + 1; i < len(format); i++ {
			if format[i] == '0' || format[i] == '#' {
				decimals++
			} else {
				break
			}
		}
	}

	// Round to decimals
	mult := math.Pow(10, float64(decimals))
	rounded := math.Round(num*mult) / mult

	negative := rounded < 0
	if negative {
		rounded = -rounded
	}

	intPart := int64(rounded)
	fracPart := rounded - float64(intPart)

	var result strings.Builder

	// Format integer part
	intStr := strconv.FormatInt(intPart, 10)
	if hasComma {
		for i, ch := range intStr {
			if i > 0 && (len(intStr)-i)%3 == 0 {
				result.WriteByte(',')
			}
			result.WriteRune(ch)
		}
	} else {
		result.WriteString(intStr)
	}

	// Format decimal part
	if decimals > 0 {
		fracStr := fmt.Sprintf("%.*f", decimals, fracPart)
		result.WriteString(fracStr[1:]) // Skip the "0"
	}

	if negative {
		return "-" + result.String()
	}
	return result.String()
}
