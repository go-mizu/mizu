package str_test

import (
	"fmt"
	"unicode/utf8"

	"github.com/go-mizu/mizu/str"
)

func ExampleGraphemes() {
	// The n is followed by a combining tilde, so it is two code points and one
	// character. The flag is two code points as well.
	for g := range str.Graphemes("an\u0303\U0001F1EF\U0001F1F5") {
		fmt.Printf("%d bytes, %d runes\n", len(g), utf8.RuneCountInString(g))
	}
	// Output:
	// 1 bytes, 1 runes
	// 3 bytes, 2 runes
	// 8 bytes, 2 runes
}

func ExampleLength() {
	// The flag is two code points and eight bytes, and one character.
	fmt.Println(str.Length("🇯🇵"))
	fmt.Println(str.Length("hello"))
	// Output:
	// 1
	// 5
}

func ExampleCamel() {
	fmt.Println(str.Camel("user_id"))
	fmt.Println(str.Camel("HTTP server"))
	// Output:
	// userId
	// httpServer
}

func ExamplePascal() {
	fmt.Println(str.Pascal("order_line_item"))
	// Output: OrderLineItem
}

func ExampleSnake() {
	fmt.Println(str.Snake("userID"))
	fmt.Println(str.Snake("HTTPServer"))
	// Output:
	// user_id
	// http_server
}

func ExampleKebab() {
	fmt.Println(str.Kebab("BackgroundColor"))
	// Output: background-color
}

func ExampleHeadline() {
	fmt.Println(str.Headline("steve_jobs"))
	fmt.Println(str.Headline("email_notification_sent"))
	// Output:
	// Steve Jobs
	// Email Notification Sent
}

func ExampleTitle() {
	fmt.Println(str.Title("a nice day out"))
	// Output: A Nice Day Out
}

func ExampleSentence() {
	fmt.Println(str.Sentence("hello world. we ship on Tuesday."))
	// Output: Hello world. We ship on Tuesday.
}

func ExampleUpperFirst() {
	fmt.Println(str.UpperFirst("hello world"))
	// Output: Hello world
}

func ExampleLowerFirst() {
	fmt.Println(str.LowerFirst("Hello World"))
	// Output: hello World
}

func ExampleSwapCase() {
	fmt.Println(str.SwapCase("Hello World"))
	// Output: hELLO wORLD
}

func ExampleBefore() {
	fmt.Println(str.Before("user@example.com", "@"))
	// Nothing to find, so the whole string comes back.
	fmt.Println(str.Before("no sign here", "@"))
	// Output:
	// user
	// no sign here
}

func ExampleBeforeLast() {
	fmt.Println(str.BeforeLast("a/b/c.txt", "/"))
	// Output: a/b
}

func ExampleAfter() {
	fmt.Println(str.After("user@example.com", "@"))
	// Output: example.com
}

func ExampleAfterLast() {
	fmt.Println(str.AfterLast("/var/log/app.log", "/"))
	fmt.Println(str.AfterLast("archive.tar.gz", "."))
	// Output:
	// app.log
	// gz
}

func ExampleBetween() {
	// Between reaches for the last closing mark, so it takes in the pair in the
	// middle rather than stopping at the first one.
	fmt.Println(str.Between("[a] and [b]", "[", "]"))
	fmt.Println(str.BetweenFirst("[a] and [b]", "[", "]"))
	// Output:
	// a] and [b
	// a
}

func ExampleSubstr() {
	fmt.Println(str.Substr("hello world", 6, 5))
	fmt.Println(str.Substr("hello world", -5, 5))
	// A negative length drops that many from the end instead.
	fmt.Println(str.Substr("hello world", 1, -1))
	// Output:
	// world
	// world
	// ello worl
}

func ExampleTake() {
	fmt.Println(str.Take("hello world", 5))
	fmt.Println(str.Take("hello world", -5))
	// Output:
	// hello
	// world
}

func ExampleLimit() {
	fmt.Println(str.Limit("the quick brown fox", 9, "..."))
	// Short enough already, so nothing is added.
	fmt.Println(str.Limit("short", 9, "..."))
	// Output:
	// the quick...
	// short
}

func ExampleWords() {
	fmt.Println(str.Words("the quick brown fox jumps", 3, "..."))
	// Output: the quick brown...
}

func ExampleExcerpt() {
	fmt.Println(str.Excerpt("the quick brown fox jumps", "brown", 6, "..."))
	// A phrase that is not there gives nothing rather than everything.
	fmt.Printf("%q\n", str.Excerpt("the quick brown fox jumps", "purple", 6, "..."))
	// Output:
	// ...quick brown fox j...
	// ""
}

func ExampleMask() {
	fmt.Println(str.Mask("4111111111111111", '*', 4))
	// Asking to keep more than is there leaves the string alone rather than
	// showing part of it.
	fmt.Println(str.Mask("1234", '*', 8))
	// Output:
	// ************1111
	// 1234
}

func ExampleReverse() {
	fmt.Println(str.Reverse("hello"))

	// The e carries a combining acute after it, and the accent stays on the
	// letter it belongs to rather than sliding onto the one next door.
	fmt.Println(str.Reverse("cafe\u0301") == "e\u0301fac")
	// Output:
	// olleh
	// true
}

func ExamplePadLeft() {
	fmt.Println(str.PadLeft("7", 3, "0"))
	// Output: 007
}

func ExamplePadRight() {
	fmt.Println(str.PadRight("ana", 6, "."))
	// Output: ana...
}

func ExamplePad() {
	fmt.Println(str.Pad("ana", 9, "-"))
	// An odd amount of padding puts the extra one on the right.
	fmt.Println(str.Pad("ana", 8, "-"))
	// Output:
	// ---ana---
	// --ana---
}

func ExampleWrap() {
	fmt.Println(str.Wrap("value", `"`, `"`))
	fmt.Println(str.Wrap("b", "<i>", "</i>"))
	// Output:
	// "value"
	// <i>b</i>
}

func ExampleUnwrap() {
	fmt.Println(str.Unwrap(`"value"`, `"`, `"`))
	// Only one end matches, so nothing is taken off.
	fmt.Println(str.Unwrap(`"value`, `"`, `"`))
	// Output:
	// value
	// "value
}
