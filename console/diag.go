package console

import (
	"bytes"
	"os"

	"github.com/go-mizu/mizu/errs/diag"
)

// fail reports the error that ended a command.
//
// Under --json it is the mizu.diag/1 document from doc 37 section 4.2, and it
// goes to stderr rather than to stdout. stdout is the answer the command was
// asked for and it is one document: a command that had already written half of
// one before it failed would otherwise produce a stream that is not JSON at
// all. mizu doctor --json --ci is exactly that command. stderr is what went
// wrong, also one document, and neither can corrupt the other.
//
// A command that returned an ordinary error still gets a document, because
// [diag.Of] makes one out of anything. That is what makes --json true of every
// command rather than of the ones that were written with it in mind.
func (c *IO) fail(err error) {
	if !c.jsonMode {
		c.Error("%v", err)
		return
	}
	if werr := diag.JSON(c.err, diag.Of(err)); werr != nil {
		// Nothing left to report it with, so say it the way a person would
		// read. A stderr that will not take a JSON document will not take a
		// second attempt at one either.
		c.Error("%v", err)
	}
}

// writeDiag writes the mizu.diag/1 document to the path --diag-file named.
//
// It is doc 36 section 2.4. A generator invoked through go generate has no
// command line to put --json on, and its stdout belongs to the file it is
// writing, so the document goes to the path in MIZU_DIAG_FILE instead.
//
// It is written on every run, including one that found nothing, where it is an
// empty list and a summary of zeroes. Telling an empty run from one that never
// started should not depend on whether there was any output.
//
// The document is built before the file is opened, so what lands on disk is the
// whole of it or nothing. A half written file is not JSON, and a reader has no
// way to tell that from a run that crashed partway.
//
// A file that cannot be written is a warning rather than an error. The command
// has already finished by the time this runs, and taking its exit code away now
// would say the work did not happen when it did.
func (c *IO) writeDiag(err error) {
	var b bytes.Buffer
	werr := diag.JSON(&b, diag.Of(err))
	if werr == nil {
		werr = os.WriteFile(c.diagFile, b.Bytes(), 0o644)
	}
	if werr != nil {
		c.Warn("--diag-file: %v", werr)
	}
}
