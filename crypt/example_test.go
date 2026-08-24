package crypt_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-mizu/mizu/crypt"
	"github.com/go-mizu/mizu/log"
)

func ExampleGenerateKey() {
	key := crypt.GenerateKey()

	// Reveal is the text to put in the environment. Everything else about the
	// key is redacted, including printing it.
	text := key.Reveal()
	fmt.Println(strings.HasPrefix(text, "mizu1:"), len(text))
	fmt.Println(key)

	// Output:
	// true 49
	// mizu1:[redacted]
}

func ExampleParseKey() {
	key, err := crypt.ParseKey(os.Getenv("APP_KEY"))
	if err != nil {
		fmt.Println("APP_KEY:", err)
		return
	}
	fmt.Println(key.ID())
}

// ExampleKey_ID shows the part of a key that is safe to write down. It names
// the key a ciphertext belongs to, without saying anything about the key.
func ExampleKey_ID() {
	key := crypt.MustParseKey("mizu1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println(key.ID())

	// Output:
	// 2d0845ee2746f8e8
}

// ExampleSecret is the point of the type: the masking travels with the value,
// so a configuration struct is safe to print and safe to log.
func ExampleSecret() {
	cfg := struct {
		URL   string
		Token crypt.Secret
	}{
		URL:   "https://api.example.com",
		Token: "t0ps3cr3t-not-a-real-token",
	}

	fmt.Printf("%+v\n", cfg)
	fmt.Println(cfg.Token.Reveal()[:9] + "...")

	// Output:
	// {URL:https://api.example.com Token:[redacted]}
	// t0ps3cr3t...
}

func ExampleSecret_Equal() {
	const want = crypt.Secret("t0ps3cr3t")

	// Constant time, because comparing with == takes longer the more leading
	// bytes match, and a caller that gets to keep asking can use that.
	fmt.Println(want.Equal("t0ps3cr3t"))
	fmt.Println(want.Equal("t0ps3cr3u"))

	// Output:
	// true
	// false
}

func ExampleToken() {
	// 32 bytes of entropy, as 43 URL safe characters.
	id := crypt.Token(32)
	fmt.Println(len(id), strings.ContainsAny(id, "+/="))

	// Output:
	// 43 false
}

func ExampleDigits() {
	code := crypt.Digits(6)
	fmt.Println(len(code), strings.Trim(code, "0123456789") == "")

	// Output:
	// 6 true
}

// ExampleCrypt is the whole of it: a key in, a ciphertext out, the message back.
func ExampleCrypt() {
	c, err := crypt.New(crypt.GenerateKey())
	if err != nil {
		fmt.Println(err)
		return
	}

	b := c.Encrypt([]byte("4111 1111 1111 1111"))
	fmt.Println(len(b) == len("4111 1111 1111 1111")+crypt.Overhead)

	card, err := c.Decrypt(b)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s\n", card)

	// Output:
	// true
	// 4111 1111 1111 1111
}

// ExampleAD shows what binding buys: the same ciphertext, handed over for
// another user, does not open.
func ExampleAD() {
	c, err := crypt.New(crypt.GenerateKey())
	if err != nil {
		fmt.Println(err)
		return
	}

	b := c.Encrypt([]byte("4111 1111 1111 1111"), crypt.AD("user:42"))

	_, err = c.Decrypt(b, crypt.AD("user:42"))
	fmt.Println("owner:", err)

	_, err = c.Decrypt(b, crypt.AD("user:43"))
	fmt.Println("anybody else:", err)

	// Output:
	// owner: <nil>
	// anybody else: crypt: this ciphertext does not open with the key it names
}

// ExampleCrypt_EncryptString is the form for a value that has to survive a
// cookie or a URL.
func ExampleCrypt_EncryptString() {
	c, err := crypt.New(crypt.GenerateKey())
	if err != nil {
		fmt.Println(err)
		return
	}

	token := c.EncryptString("user:42", crypt.AD("session"))
	fmt.Println(strings.ContainsAny(token, "+/="))

	who, err := c.DecryptString(token, crypt.AD("session"))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(who)

	// Output:
	// false
	// user:42
}

// ExampleCrypt_Rotate is a key being replaced while the application is running.
// What was written under the old key keeps opening, and NeedsRewrap is how a job
// finds it.
func ExampleCrypt_Rotate() {
	c, err := crypt.New(crypt.GenerateKey())
	if err != nil {
		fmt.Println(err)
		return
	}
	old := c.Encrypt([]byte("4111 1111 1111 1111"))

	c, err = c.Rotate(crypt.GenerateKey())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("needs rewrap:", c.NeedsRewrap(old))

	card, err := c.Decrypt(old)
	if err != nil {
		fmt.Println(err)
		return
	}
	fresh := c.Encrypt(card)
	fmt.Println("needs rewrap:", c.NeedsRewrap(fresh))

	// Output:
	// needs rewrap: true
	// needs rewrap: false
}

// ExampleCrypt_Sign is a value that stays readable and cannot be changed, which
// is what an unsubscribe link or a user id in a cookie wants.
func ExampleCrypt_Sign() {
	c, err := crypt.New(crypt.GenerateKey())
	if err != nil {
		fmt.Println(err)
		return
	}

	b := c.Sign([]byte("user:42"))
	fmt.Println(bytes.Contains(b, []byte("user:42")))

	who, err := c.Verify(b)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s\n", who)

	// Somebody who edits the message and hands it back gets nothing.
	forged := bytes.Replace(b, []byte("42"), []byte("43"), 1)
	_, err = c.Verify(forged)
	fmt.Println("forged:", err)

	// Output:
	// true
	// user:42
	// forged: crypt: this value does not carry a tag from the key it names
}

// ExampleSeal is a value in and a value out, with the encoding and the
// encryption in between.
func ExampleSeal() {
	type Session struct {
		User  string `json:"user"`
		Admin bool   `json:"admin"`
	}

	c, err := crypt.New(crypt.GenerateKey())
	if err != nil {
		fmt.Println(err)
		return
	}

	token, err := crypt.Seal(c, Session{User: "42", Admin: true}, crypt.AD("session"))
	if err != nil {
		fmt.Println(err)
		return
	}

	s, err := crypt.Unseal[Session](c, token, crypt.AD("session"))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", s)

	// The same token read as something else, which is what a cookie from before
	// a deploy looks like.
	_, err = crypt.Unseal[[]string](c, token, crypt.AD("session"))
	fmt.Println("as a list:", err != nil)

	// Output:
	// {User:42 Admin:true}
	// as a list: true
}

// Example_logging shows what a handler does with these types. Both of them
// carry their own masking, so nothing has to be configured for this to hold.
func Example_logging() {
	logger := slog.New(log.NewConsoleHandler(os.Stdout, log.ConsoleOptions{
		TimeFormat: "-",
		Color:      log.ColorNever,
		MsgWidth:   1,
	}))

	logger.Info("configured",
		"key", crypt.MustParseKey("mizu1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
		"webhook", crypt.Secret("t0ps3cr3t-not-a-real-token"),
	)

	// Output:
	// - INF configured key=mizu1:[redacted] webhook=[redacted]
}
