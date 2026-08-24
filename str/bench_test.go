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

func BenchmarkBefore(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Before("user@example.com", "@")
	}
}

func BenchmarkAfterLast(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.AfterLast("/var/log/mizu/app.log", "/")
	}
}

func BenchmarkBetween(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Between("<title>the page</title>", "<title>", "</title>")
	}
}

func BenchmarkSubstrASCII(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Substr(plainASCII, 100, 50)
	}
}

func BenchmarkSubstrEmoji(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Substr(emoji, 10, 20)
	}
}

func BenchmarkLimit(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Limit(plainASCII, 80, "...")
	}
}

func BenchmarkWords(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Words(plainASCII, 20, "...")
	}
}

func BenchmarkExcerpt(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Excerpt(plainASCII, "brown", 40, "...")
	}
}

func BenchmarkMask(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Mask("4111111111111111", '*', 4)
	}
}

func BenchmarkReverseASCII(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Reverse(plainASCII)
	}
}

func BenchmarkReverseEmoji(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Reverse(emoji)
	}
}

func BenchmarkPadLeft(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.PadLeft("42", 12, "0")
	}
}

func BenchmarkPad(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Pad("title", 40, "-")
	}
}

// BenchmarkTakeFromALongString and BenchmarkTakeFromAShortString are here as a
// pair, because what matters is that they cost about the same. Taking the first
// few characters walks those few characters and stops, so the length of the
// string behind them does not come into it.
func BenchmarkTakeFromALongString(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Take(plainASCII, 5)
	}
}

func BenchmarkTakeFromAShortString(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkStr = str.Take("the quick", 5)
	}
}
