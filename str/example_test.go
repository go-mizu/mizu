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
