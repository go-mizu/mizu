package str_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/go-mizu/mizu/str"
)

var (
	sinkInt int
	sinkStr string
)

// The three shapes of text worth measuring separately, because the segmenter
// takes a different path through the tables for each of them.
var (
	plainASCII = strings.Repeat("the quick brown fox ", 50)
	accented   = strings.Repeat("le vif renard brun é ", 50)
	emoji      = strings.Repeat("👨‍👩‍👧‍👦🇯🇵👍🏽", 50)
)

func BenchmarkLengthASCII(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkInt = str.Length(plainASCII)
	}
}

// BenchmarkRuneCountASCII is the standard library counting the same string, so
// the price of knowing where the characters really are is visible.
func BenchmarkRuneCountASCII(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkInt = utf8.RuneCountInString(plainASCII)
	}
}

func BenchmarkLengthAccented(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkInt = str.Length(accented)
	}
}

func BenchmarkLengthEmoji(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkInt = str.Length(emoji)
	}
}

func BenchmarkGraphemes(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		n := 0
		for range str.Graphemes(accented) {
			n++
		}
		sinkInt = n
	}
}

func BenchmarkCamel(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Camel("background_color_override")
	}
}

func BenchmarkSnake(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Snake("BackgroundColorOverride")
	}
}

func BenchmarkHeadline(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Headline("email_notification_sent_to_user")
	}
}

func BenchmarkTitle(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Title("the quick brown fox jumps over the lazy dog")
	}
}

func BenchmarkUpperFirst(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.UpperFirst("hello world")
	}
}

func BenchmarkSwapCase(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.SwapCase(plainASCII)
	}
}
