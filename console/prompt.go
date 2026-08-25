package console

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// Interaction is whether a command may stop and ask a question.
//
// It is a mode rather than a bool because there are three answers, and the
// interesting one is the default: ask when there is somebody there to answer.
type Interaction int

const (
	// InteractionAuto asks when stdin and stderr are both terminals. It is the
	// zero value and it is what a program should use unless a flag said
	// otherwise.
	InteractionAuto Interaction = iota

	// InteractionNever is --no-interaction. Every prompt takes its default or
	// fails.
	InteractionNever

	// InteractionAlways asks whatever the streams are. This is what a test
	// uses, since a buffer holding scripted answers is not a terminal.
	InteractionAlways
)

// String returns the name the flag uses.
func (i Interaction) String() string {
	switch i {
	case InteractionNever:
		return "never"
	case InteractionAlways:
		return "always"
	default:
		return "auto"
	}
}

// ErrAborted is returned by a prompt the user walked away from, with Ctrl-C or
// Ctrl-D or by closing the stream it was reading.
//
// It is a plain answer to a question, not a failure of the command, so a
// command line program returning it exits 130 and prints nothing.
var ErrAborted = errors.New("aborted")

// ErrNoInput is what a prompt that cannot be asked and has no default returns,
// wrapped in an error naming the question.
//
// This is the rule that keeps a command from hanging in CI forever. A build
// that reaches a question nobody can answer stops with a sentence about the
// missing value instead of holding a runner until it times out.
var ErrNoInput = errors.New("there is no terminal and no default, so the value has to come from a flag")

// noInput returns the error a prompt gives when it cannot be asked.
//
// The message would be better if it named the flag, which is what the rule in
// the specification asks for. This package does not know the name: the value
// and the flag are tied together in the command struct, which is not written
// yet. The wrapping is here so that the layer which does know can add it.
func noInput(question string) error {
	return fmt.Errorf("cannot ask %q: %w", question, ErrNoInput)
}

// Interactive reports whether prompts will ask rather than take their default.
//
// A command uses it to skip work that only makes sense with somebody watching,
// such as offering an optional list to choose from. Prompts themselves do not
// need the caller to check.
func (c *IO) Interactive() bool { return c.interactive }

// canPrompt decides whether an IO may ask questions.
//
// Both streams have to be terminals. stdin being one means there is somebody
// typing, and stderr being one means they can see what is being asked. A
// program reading a script from a pipe with its stderr on a terminal has half
// of a conversation, and half is worse than none.
func canPrompt(in io.Reader, errw io.Writer, mode Interaction) bool {
	switch mode {
	case InteractionNever:
		return false
	case InteractionAlways:
		return true
	}
	return isTerminal(in) && isTerminal(errw)
}

// Ask asks for a line of text, returning def when the answer is empty.
//
// Leading and trailing spaces are dropped, because at a prompt they are a slip
// rather than an answer. Use [IO.AskSecret] for anything where they might not
// be.
//
// With no terminal, an empty def is an error. There is nothing to fall back on
// and no way to find out, and the alternative is a command that quietly does
// its work with an empty string in the middle of it.
func (c *IO) Ask(question, def string) (string, error) {
	if !c.interactive {
		if def == "" {
			return "", noInput(question)
		}
		return def, nil
	}

	c.ask(question, def)
	line, err := c.readLine()
	if err != nil {
		return "", err
	}
	if answer := strings.TrimSpace(line); answer != "" {
		return answer, nil
	}
	return def, nil
}

// AskSecret asks for a line of text without echoing it.
//
// The answer is returned exactly as typed, spaces and all, since a password
// with a space at the end of it is still that password.
//
// There is no default. A secret nobody typed is not a secret, so with no
// terminal this is always an error.
func (c *IO) AskSecret(question string) (string, error) {
	if !c.interactive {
		return "", noInput(question)
	}

	c.ask(question, "")
	n, isatty := fd(c.in)
	if !isatty || !term.IsTerminal(n) {
		// Not a terminal, which under InteractionAlways means a test or a
		// script. There is no echo to turn off, so read the line and take the
		// caller at their word about what they piped in.
		return c.readLine()
	}

	// The newline the user pressed is consumed by the terminal and never
	// echoed, so without this the next thing written lands on the prompt.
	secret, err := term.ReadPassword(n)
	fmt.Fprintln(c.err)
	if err != nil {
		return "", readError(err)
	}
	return string(secret), nil
}

// Confirm asks a yes or no question.
//
// An empty answer takes def, and so does the whole prompt when there is no
// terminal. That is deliberate and it is why the argument exists: a command
// guarding something destructive passes false, so the same command in CI stops
// rather than proceeds.
func (c *IO) Confirm(question string, def bool) (bool, error) {
	if !c.interactive {
		return def, nil
	}

	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	for {
		c.ask(question, hint)
		line, err := c.readLine()
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "":
			return def, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
		c.say(styleYellow, "", "answer y or n")
	}
}

// Choice asks for one of options, given as a numbered list, and returns its
// index. A def in range is what an empty answer takes.
//
// The list is numbered rather than driven by the arrow keys. A number is
// something a person can say out loud to somebody else, it survives a terminal
// that does not report key presses the way this one expects, and it is the
// difference between a prompt that works over a serial console and one that
// does not.
func (c *IO) Choice(question string, options []string, def int) (int, error) {
	if len(options) == 0 {
		return 0, errors.New("a choice with nothing to choose from")
	}
	hasDefault := def >= 0 && def < len(options)

	if !c.interactive {
		if !hasDefault {
			return 0, noInput(question)
		}
		return def, nil
	}

	c.list(options)
	hint := ""
	if hasDefault {
		hint = strconv.Itoa(def + 1)
	}
	for {
		c.ask(question, hint)
		line, err := c.readLine()
		if err != nil {
			return 0, err
		}
		answer := strings.TrimSpace(line)
		if answer == "" && hasDefault {
			return def, nil
		}
		if i, err := index(answer, len(options)); err == nil {
			return i, nil
		}
		c.say(styleYellow, "", "answer with a number from 1 to %d", len(options))
	}
}

// MultiChoice asks for any number of options and returns their indexes, in the
// order the list was given rather than the order they were typed.
//
// An empty answer selects nothing, which is a real answer. There is no default
// argument for the same reason: with no terminal this is an error, so that a
// scaffolder in CI says which flag it wanted instead of generating a project
// with nothing in it. A command that wants to go on regardless checks
// [IO.Interactive] first.
func (c *IO) MultiChoice(question string, options []string) ([]int, error) {
	if len(options) == 0 {
		return nil, errors.New("a choice with nothing to choose from")
	}
	if !c.interactive {
		return nil, noInput(question)
	}

	c.list(options)
	c.say(styleDim, "", "numbers separated by commas, or nothing for none")
	for {
		c.ask(question, "")
		line, err := c.readLine()
		if err != nil {
			return nil, err
		}
		picked, err := indexes(strings.TrimSpace(line), len(options))
		if err == nil {
			return picked, nil
		}
		c.say(styleYellow, "", "answer with numbers from 1 to %d, separated by commas", len(options))
	}
}

// index reads one entry number, one based, and returns its index.
func index(s string, n int) (int, error) {
	i, err := strconv.Atoi(s)
	if err != nil || i < 1 || i > n {
		return 0, fmt.Errorf("not a number from 1 to %d", n)
	}
	return i - 1, nil
}

// indexes reads a comma separated list of entry numbers. Repeats collapse and
// the result comes back in list order, so that 3,1,1 and 1,3 are one answer.
func indexes(s string, n int) ([]int, error) {
	if s == "" {
		return nil, nil
	}
	var picked []int
	for _, field := range strings.Split(s, ",") {
		i, err := index(strings.TrimSpace(field), n)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(picked, i) {
			picked = append(picked, i)
		}
	}
	slices.Sort(picked)
	return picked, nil
}

// ask writes the question to stderr and leaves the cursor after it.
func (c *IO) ask(question, hint string) {
	if hint != "" {
		question += " [" + hint + "]"
	}
	fmt.Fprint(c.err, styleBold.wrap(question, c.colorErr), ": ")
}

// list writes the numbered options to stderr.
func (c *IO) list(options []string) {
	width := len(strconv.Itoa(len(options)))
	for i, option := range options {
		fmt.Fprintf(c.err, "  %*d  %s\n", width, i+1, option)
	}
}

// readLine reads one line from stdin, without the line ending.
//
// The reader is kept on the IO because it buffers. Building a new one per
// prompt would read ahead and throw away whatever the next prompt was going to
// see, which shows up as a second question the user never got to answer.
func (c *IO) readLine() (string, error) {
	if c.reader == nil {
		c.reader = bufio.NewReader(c.in)
	}
	line, err := c.reader.ReadString('\n')
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	if err != nil {
		// A last line with nothing after it is still an answer. The stream
		// ending is only an abort when there was nothing on it.
		if errors.Is(err, io.EOF) && line != "" {
			return line, nil
		}
		return "", readError(err)
	}
	return line, nil
}

// readError turns the end of the input into [ErrAborted] and leaves anything
// else alone, since a broken pipe and a user pressing Ctrl-D are different
// things to report.
func readError(err error) error {
	if errors.Is(err, io.EOF) {
		return ErrAborted
	}
	return err
}
