package se

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/pkg/password"
	"github.com/go-mizu/mizu/blueprints/qa/pkg/text"
)

const (
	maxUsernameLen = 20
)

var defaultPasswordHash = mustHashPassword("imported")

func mustHashPassword(passwordRaw string) string {
	hash, err := password.Hash(passwordRaw)
	if err != nil {
		panic(err)
	}
	return hash
}

func userID(id int64) string {
	return fmt.Sprintf("se-user-%d", id)
}

func userUsername(id int64, displayName string) string {
	if id == 0 {
		return "community"
	}
	base := sanitizeUsername(displayName)
	if base == "" {
		return fallbackUsername(id)
	}
	suffix := fmt.Sprintf("_%d", id)
	if len(base)+len(suffix) > maxUsernameLen {
		trimmedLen := maxUsernameLen - len(suffix)
		if trimmedLen <= 0 {
			return fallbackUsername(id)
		}
		base = strings.Trim(base[:trimmedLen], "_")
		if base == "" {
			return fallbackUsername(id)
		}
	}
	username := base + suffix
	if len(username) < 3 {
		return fallbackUsername(id)
	}
	return username
}

func sanitizeUsername(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")

	var out strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			out.WriteRune(r)
		}
	}
	cleaned := strings.Trim(out.String(), "_")
	if len(cleaned) > maxUsernameLen {
		cleaned = strings.TrimRight(cleaned[:maxUsernameLen], "_")
	}
	return cleaned
}

func fallbackUsername(id int64) string {
	base := fmt.Sprintf("user%d", id)
	if len(base) <= maxUsernameLen {
		return base
	}
	suffixLen := maxUsernameLen - 4
	if suffixLen <= 0 {
		return base[:maxUsernameLen]
	}
	if len(base) > suffixLen {
		return "user" + base[len(base)-suffixLen:]
	}
	return base
}

func userEmail(id int64) string {
	return fmt.Sprintf("user%d@se.local", id)
}

func questionID(id int64) string {
	return fmt.Sprintf("se-question-%d", id)
}

func answerID(id int64) string {
	return fmt.Sprintf("se-answer-%d", id)
}

func commentID(id int64) string {
	return fmt.Sprintf("se-comment-%d", id)
}

func voteID(id int64) string {
	return fmt.Sprintf("se-vote-%d", id)
}

func bookmarkID(id int64) string {
	return fmt.Sprintf("se-bookmark-%d", id)
}

func badgeID(name string) string {
	return "se-badge-" + url.PathEscape(strings.ToLower(name))
}

func badgeAwardID(id int64) string {
	return fmt.Sprintf("se-badge-award-%d", id)
}

func tagID(name string) string {
	return "se-tag-" + url.PathEscape(strings.ToLower(name))
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	layouts := []string{
		"2006-01-02T15:04:05.999",
		"2006-01-02T15:04:05.99",
		"2006-01-02T15:04:05.9",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return t
		}
	}
	return time.Time{}
}

func parseTags(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	var tags []string
	for {
		start := strings.IndexByte(value, '<')
		if start == -1 {
			break
		}
		end := strings.IndexByte(value[start+1:], '>')
		if end == -1 {
			break
		}
		end = start + 1 + end
		tag := strings.ToLower(value[start+1 : end])
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
		value = value[end+1:]
	}
	return tags
}

func stripHTML(value string) string {
	return text.StripHTML(value)
}

type postType int64

const (
	postTypeQuestion   postType = 1
	postTypeAnswer     postType = 2
	postTypeTagWiki    postType = 4
	postTypeTagExcerpt postType = 5
)

const (
	voteTypeUpvote   int64 = 2
	voteTypeDownvote int64 = 3
	voteTypeFavorite int64 = 5
)

func voteValue(voteTypeID int64) (int, bool) {
	switch voteTypeID {
	case voteTypeUpvote:
		return 1, true
	case voteTypeDownvote:
		return -1, true
	default:
		return 0, false
	}
}

func badgeTier(class int64) string {
	switch class {
	case 1:
		return "gold"
	case 2:
		return "silver"
	case 3:
		return "bronze"
	default:
		return "bronze"
	}
}
