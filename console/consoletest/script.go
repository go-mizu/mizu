package consoletest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

// A Prompt is a question a command asked, as the script saw it.
type Prompt struct {
	// Question is what was asked, without the default in brackets and without
	// the colon after it.
	Question string

	// Hint is what was in the brackets, which is the default for a question
	// that has one and "y/N" or "Y/n" for a confirmation. It is empty when
	// there was nothing there.
	Hint string

	// Options are the numbered choices printed above the question, in the
	// order they were offered, for a prompt that had a list.
	Options []string
}

// String returns the prompt the way it appeared on the screen.
func (p Prompt) String() string {
	if p.Hint == "" {
		return p.Question
	}
	return p.Question + " [" + p.Hint + "]"
}

// A step is one scripted answer, and the question it expects to answer.
type step struct {
	want  string
	reply func(Prompt) (string, error)
}

// script is the stdin a command under test reads.
//
// It answers a question by reading it off the stream the question was written
// to, which is what makes a scripted answer match the prompt it belongs to
// rather than the position it happens to be in.
type script struct {
	out   *bytes.Buffer // where the questions are written
	steps []step

	at    int    // the next step to use
	seen  int    // how much of out has been taken as a question already
	typed []byte // what is left of the answer being handed over

	asked   []Prompt
	problem error
}

// Read hands over the next scripted answer, one line at a time.
func (s *script) Read(p []byte) (int, error) {
	if len(s.typed) == 0 {
		reply, err := s.next()
		if err != nil {
			return 0, err
		}
		s.typed = []byte(reply + "\n")
	}
	n := copy(p, s.typed)
	s.typed = s.typed[n:]
	return n, nil
}

// next takes the question the command has just written and returns the answer
// the script has for it.
func (s *script) next() (string, error) {
	prompt := s.pending()
	s.asked = append(s.asked, prompt)

	if s.at >= len(s.steps) {
		return "", s.stop("the command asked %q and the script has no answer for it", prompt.String())
	}
	next := s.steps[s.at]
	s.at++

	if !strings.Contains(prompt.Question, next.want) {
		return "", s.stop("the script answers %q and the question was %q", next.want, prompt.Question)
	}
	reply, err := next.reply(prompt)
	if err != nil {
		return "", s.stop("the question %q %v", prompt.Question, err)
	}
	return reply, nil
}

// stop records why the script could not go on and ends the stream, which is
// what a person walking away from a prompt does.
//
// The command unwinds with [github.com/go-mizu/mizu/console.ErrAborted], the
// same as it would in a terminal, and the test reports the real reason once the
// command has finished rather than from inside a Read that nobody is watching.
func (s *script) stop(format string, a ...any) error {
	if s.problem == nil {
		s.problem = fmt.Errorf(format, a...)
	}
	return io.EOF
}

// pending reads the question off the stream it was written to.
//
// A prompt writes the question and then reads, so whatever is on the stream
// after the last newline is the question waiting for an answer, and the
// numbered lines above it are the list to choose from. Working it out this way
// is what lets the console package stay as it is, with no hook in it that only
// a test would use.
func (s *script) pending() Prompt {
	written := s.out.String()
	block := written[s.seen:]
	s.seen = len(written)

	line := block
	if i := strings.LastIndexByte(line, '\n'); i >= 0 {
		line = line[i+1:]
	}
	line = strings.TrimSuffix(stripANSI(line), ": ")

	p := Prompt{Question: line, Options: options(block)}
	if rest, ok := strings.CutSuffix(p.Question, "]"); ok {
		if i := strings.LastIndexByte(rest, '['); i > 0 {
			p.Question, p.Hint = strings.TrimSpace(rest[:i]), rest[i+1:]
		}
	}
	return p
}

// options are the numbered choices in a block of prompt output.
//
// The numbering has to run from one without a gap, so that a line of the
// command's own output that happens to start with a number does not become a
// choice nobody offered.
func options(block string) []string {
	var out []string
	for line := range strings.Lines(block) {
		n, text, ok := option(line)
		if ok && n == len(out)+1 {
			out = append(out, text)
		}
	}
	return out
}

// option reads one line of a numbered list, as [console.IO.Choice] writes it:
// some spaces, the number, two spaces, and the text.
func option(line string) (n int, text string, ok bool) {
	rest := strings.TrimLeft(stripANSI(line), " ")
	digits := 0
	for digits < len(rest) && rest[digits] >= '0' && rest[digits] <= '9' {
		digits++
	}
	if digits == 0 || !strings.HasPrefix(rest[digits:], "  ") {
		return 0, "", false
	}
	n, err := strconv.Atoi(rest[:digits])
	if err != nil {
		return 0, "", false
	}
	return n, strings.TrimRight(rest[digits+2:], "\r\n"), true
}

// stripANSI removes the escape sequences a coloured prompt carries, so that a
// test which turned colour on matches the same question as one that did not.
func stripANSI(s string) string {
	if !strings.Contains(s, "\x1b[") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < '@' || s[j] > '~') {
				j++
			}
			i = min(j+1, len(s))
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// Answer types a line at the next question.
//
// question is matched against what the command asked, which is the question
// alone: not the default in brackets, not the colon, and not anything else on
// the screen. Any part of it will do, so that a question with a count in it can
// be matched without the test working the count out first.
//
//	consoletest.Answer("Delete", "yes")
//
// An empty reply is the user pressing enter, which takes the default.
func Answer(question, reply string) Option {
	return answer(question, func(Prompt) (string, error) { return reply, nil })
}

// Confirm answers the next yes or no question.
//
// It checks that the question is one, so a script that has fallen out of step
// with the command says so rather than typing y at something that wanted a
// name.
func Confirm(question string, yes bool) Option {
	return answer(question, func(p Prompt) (string, error) {
		if p.Hint != "y/N" && p.Hint != "Y/n" {
			return "", errors.New("is not a yes or no question")
		}
		if yes {
			return "y", nil
		}
		return "n", nil
	})
}

// Choose picks one option from the next numbered list, by what it says.
//
//	consoletest.Choose("Environment", "staging")
//
// The option is matched in full, since it is text the test wrote and the
// command printed rather than something a person typed. A list with no such
// option on it fails the test and names the ones there were.
func Choose(question, option string) Option {
	return answer(question, func(p Prompt) (string, error) {
		i, err := pick(p, option)
		if err != nil {
			return "", err
		}
		return strconv.Itoa(i + 1), nil
	})
}

// ChooseAll picks any number of options from the next numbered list, for a
// question asked with [console.IO.MultiChoice]. With no options it selects
// nothing, which is an answer that prompt takes.
func ChooseAll(question string, options ...string) Option {
	return answer(question, func(p Prompt) (string, error) {
		numbers := make([]string, 0, len(options))
		for _, want := range options {
			i, err := pick(p, want)
			if err != nil {
				return "", err
			}
			numbers = append(numbers, strconv.Itoa(i+1))
		}
		return strings.Join(numbers, ","), nil
	})
}

// answer adds a step to the script.
func answer(question string, reply func(Prompt) (string, error)) Option {
	return func(s *settings) {
		s.steps = append(s.steps, step{want: question, reply: reply})
	}
}

// pick is the number to type for an option, one based.
func pick(p Prompt, want string) (int, error) {
	if i := slices.Index(p.Options, want); i >= 0 {
		return i, nil
	}
	if len(p.Options) == 0 {
		return 0, fmt.Errorf("is not a list to choose from, so there is no %q to pick", want)
	}
	quoted := make([]string, len(p.Options))
	for i, o := range p.Options {
		quoted[i] = strconv.Quote(o)
	}
	return 0, fmt.Errorf("offers %s, and not %q", strings.Join(quoted, ", "), want)
}
