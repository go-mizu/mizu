package console

import (
	"bufio"
	"bytes"
	"io"
)

// indent is what one level of nesting costs. Two spaces, because a command
// that reports on four things inside three phases is already deep enough.
const indent = "  "

// Section prints a title and returns an IO whose status output sits under it.
//
// The indent applies to stderr. Data written to stdout comes out unchanged,
// because indenting it would mean a command's output changes shape depending on
// how the command chose to talk about itself, and whatever is reading the pipe
// did not ask for that.
//
// Sections nest, and a section is finished by not using it any more. There is
// nothing to close, so an early return cannot leave one open.
//
// When nobody is reading, which is --quiet and --json, the title is not printed
// and the same IO comes back. An indent around output that is not written is
// not worth an allocation.
func (c *IO) Section(title string) *IO {
	if !c.decorated() {
		return c
	}
	c.say(styleBold, "", "%s", title)

	// A prompt inside a section has to read through the same buffered reader as
	// one outside it, or whichever asks second finds its answer already eaten
	// by the other's read ahead. Building it here costs a buffer that a section
	// with no prompt in it does not use, which is the cheaper mistake.
	if c.reader == nil {
		c.reader = bufio.NewReader(c.in)
	}

	inner := *c
	inner.err = &indenter{w: c.err, prefix: indent, atLine: true}
	inner.errWidth = max(c.errWidth-len(indent), 0)
	return &inner
}

// An indenter writes a prefix at the start of every line.
//
// A carriage return counts as the start of a line, so a progress bar redrawing
// in place lands under the indent rather than back at column zero. A newline
// with nothing after it does not get one, so a blank line stays blank instead
// of becoming two spaces somebody's editor will strip.
type indenter struct {
	w      io.Writer
	prefix string
	atLine bool
}

// Write indents p and passes it on in one call, so a line written by one
// goroutine does not arrive inside a line written by another.
//
// The count returned is the length of p rather than the number of bytes that
// reached the underlying writer, since the caller is asking whether its own
// bytes were taken and the prefix is not theirs.
func (w *indenter) Write(p []byte) (int, error) {
	var b bytes.Buffer
	b.Grow(len(p) + len(w.prefix))
	for _, c := range p {
		if w.atLine && c != '\n' {
			b.WriteString(w.prefix)
			w.atLine = false
		}
		b.WriteByte(c)
		if c == '\n' || c == '\r' {
			w.atLine = true
		}
	}
	if _, err := w.w.Write(b.Bytes()); err != nil {
		return 0, err
	}
	return len(p), nil
}
