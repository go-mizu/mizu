package amazon

import (
	"crypto/md5"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseQA parses an Amazon Q&A page (/ask/<ASIN>?pageNumber=N).
// Returns all QA pairs found and the next page URL (empty if last page).
func ParseQA(doc *goquery.Document, asin, pageURL string) ([]QA, string, error) {
	var qas []QA
	now := time.Now()

	// Each question is in a div with id starting "question-"
	doc.Find(`[id^="question-"]`).Each(func(i int, s *goquery.Selection) {
		qa := QA{ASIN: asin, FetchedAt: now}
		if id, exists := s.Attr("id"); exists && strings.TrimSpace(id) != "" {
			qa.QAID = strings.TrimSpace(id)
		} else {
			qa.QAID = fmt.Sprintf("%s-%d", asin, i)
		}

		// Question text
		qa.Question = strings.TrimSpace(s.Find(".a-expander-content").First().Text())
		if qa.Question == "" {
			// fallback: first .a-declarative .a-size-base
			qa.Question = strings.TrimSpace(s.Find(".a-declarative .a-size-base").First().Text())
		}
		if qa.Question == "" {
			qa.Question = strings.TrimSpace(s.Find(".a-size-base").First().Text())
		}

		// Question author: first .a-profile-name in the question container
		qa.QuestionBy = strings.TrimSpace(s.Find(".a-profile-name").First().Text())

		// Question date: first tertiary color date-like span
		s.Find(".a-size-base.a-color-tertiary").Each(func(_ int, span *goquery.Selection) {
			if !qa.QuestionDate.IsZero() {
				return
			}
			t := strings.TrimSpace(span.Text())
			if parsed := parseQADate(t); !parsed.IsZero() {
				qa.QuestionDate = parsed
			}
		})

		// Answer: look for a sibling/child div with id starting "answer-"
		answerSel := s.Find(`[id^="answer-"]`).First()
		if answerSel.Length() == 0 {
			// answers may be at the same level in a parent container
			answerSel = s.Parent().Find(`[id^="answer-"]`).First()
		}
		if answerSel.Length() > 0 {
			qa.Answer = strings.TrimSpace(answerSel.Find(".a-expander-content").First().Text())
			if qa.Answer == "" {
				qa.Answer = strings.TrimSpace(answerSel.Find(".a-size-base").First().Text())
			}

			// Answer author: second .a-profile-name (first belongs to question)
			profiles := answerSel.Find(".a-profile-name")
			if profiles.Length() > 0 {
				qa.AnswerBy = strings.TrimSpace(profiles.First().Text())
			}

			// Answer date
			answerSel.Find(".a-size-base.a-color-tertiary").Each(func(_ int, span *goquery.Selection) {
				if !qa.AnswerDate.IsZero() {
					return
				}
				t := strings.TrimSpace(span.Text())
				if parsed := parseQADate(t); !parsed.IsZero() {
					qa.AnswerDate = parsed
				}
			})

			// IsSellerAnswer
			qa.IsSellerAnswer = answerSel.Find(`[data-hook="askSeller"]`).Length() > 0

			// HelpfulVotes from "N people found this helpful"
			answerSel.Find(".a-size-base").Each(func(_ int, span *goquery.Selection) {
				if qa.HelpfulVotes != 0 {
					return
				}
				t := strings.TrimSpace(span.Text())
				if strings.Contains(strings.ToLower(t), "found") && strings.Contains(strings.ToLower(t), "helpful") {
					qa.HelpfulVotes = int(parseInt64Digits(t))
				}
			})
		}

		if qa.Question != "" || qa.Answer != "" {
			sum := md5.Sum([]byte(asin + "|" + qa.Question + "|" + qa.Answer))
			qa.QAID = fmt.Sprintf("%x", sum)
			qas = append(qas, qa)
		}
	})

	if len(qas) == 0 {
		// Log first 500 bytes for debugging
		html, _ := doc.Html()
		limit := 500
		if len(html) < limit {
			limit = len(html)
		}
		log.Printf("ParseQA: 0 QAs parsed for ASIN %s. HTML preview: %s", asin, html[:limit])
	}

	// Next page URL
	nextPageURL := ""
	doc.Find(".a-last").Each(func(_ int, s *goquery.Selection) {
		if nextPageURL != "" {
			return
		}
		if s.HasClass("a-disabled") {
			return
		}
		href, exists := s.Find("a").Attr("href")
		if exists && href != "" {
			nextPageURL = absoluteURL(BaseURL, href)
		}
	})

	return qas, nextPageURL, nil
}

// parseQADate attempts to parse date strings commonly found in Amazon Q&A pages.
// Formats tried: "January 2, 2006", "Jan 2, 2006", "2006-01-02".
func parseQADate(text string) time.Time {
	text = strings.TrimSpace(text)
	if text == "" {
		return time.Time{}
	}
	formats := []string{
		"January 2, 2006",
		"Jan 2, 2006",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, text); err == nil {
			return t
		}
	}
	return time.Time{}
}
