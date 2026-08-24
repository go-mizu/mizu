// Package log holds [log/slog] handlers: one that writes a line a person can
// read, and one that writes a JSON object a machine can read.
//
// There is no logger type here. The logger is [slog.Logger], and everything
// that takes a *slog.Logger keeps working, including code that has never heard
// of this package. What a handler decides is what a record looks like when it
// lands, which is the part the standard library leaves open.
//
//	slog.SetDefault(slog.New(log.NewConsoleHandler(os.Stderr, log.ConsoleOptions{})))
//
// In production the same program writes JSON to standard output and lets
// whatever runs it collect the stream:
//
//	slog.SetDefault(slog.New(log.NewJSONHandler(os.Stdout, log.JSONOptions{
//		Level:     slog.LevelInfo,
//		AddSource: true,
//		Stack:     true,
//	})))
//
// # What both handlers do
//
// Records come out in a fixed order: time, level, message, source, the data on
// the context, the attributes the logger carries, and then the attributes of
// the record itself in the order they were given. A line diffs against the
// line before it, and a person reading a JSON log finds the message in the same
// place every time.
//
// An attribute holding an error is expanded rather than printed as a string.
// The message goes under the key it was given, and the kind and code from
// [github.com/go-mizu/mizu/errs] go under that key with _kind and _code after
// it, when anything actually classified the error. A plain error from a package
// that has never heard of errs is written as its message and nothing else,
// since labelling it internal would be a decision nobody made.
//
// A record at error level can also carry the stack the error was made with,
// which errs captured at the point it was created rather than at the point it
// was logged. The console prints one frame under the record. JSON writes a
// stack member when [JSONOptions.Stack] is set, and leaves it out otherwise,
// because a stack costs more to store than the rest of the record together.
//
// Data put on the context with [github.com/go-mizu/mizu/ctxdata] arrives in
// every record made under that context, so a request id or a tenant is written
// once by middleware instead of being repeated at every call.
//
// # Secrets
//
// A value that should never be printed is best as a type with a LogValue
// method, since masking that travels with the value works in every handler,
// including the ones in the standard library. Both handlers here call LogValue
// the way [log/slog] says to.
//
// Redaction by key is the backstop for values that arrive as a plain string
// from somewhere else, which in practice is where leaks come from. An attribute
// whose key is in the Redact list is written as [Mask], and the list defaults to
// [DefaultRedact]. Setting Redact to an empty non-nil slice turns it off, which
// is a thing to write down on purpose rather than to arrive at by leaving a
// field out.
//
// # Putting handlers together
//
// Three handlers take a handler and give one back, so that a policy is written
// once around whatever is doing the writing.
//
//	h = log.NewMultiHandler(h, other)   // write every record to both
//	h = log.NewFilterHandler(h, keep)   // drop the ones keep says no to
//	h = log.NewSamplingHandler(h, o)    // drop the ones that repeat
//
// [NewSamplingHandler] is the one to reach for when a message in a loop can
// fill a disk. It writes the first hundred of a message each second and then
// one in a hundred, counts them in a fixed table of counters rather than behind
// a lock, and never drops an error.
//
// # Writing to a file
//
// A program that keeps its own logs writes to a [File], which renames itself
// out of the way when it gets large, keeps the last few of the old ones and
// gzips them.
//
//	f, err := log.NewFile("/var/log/blog/app.log", log.RotateOptions{Compress: true})
//	if err != nil {
//		return err
//	}
//	defer f.Close()
//	slog.SetDefault(slog.New(log.NewJSONHandler(f, log.JSONOptions{})))
//
// It is an [io.Writer], so any handler goes on top of it, and [File.Rotate]
// starts a new file on demand for a program that is told when to.
//
// Under a process manager that already collects standard output there is
// nothing to do here. Write to os.Stdout and let it deal with the disk.
//
// # From a configuration
//
// [New] builds all of the above from a [github.com/go-mizu/mizu/config.Log],
// which is the struct an application loads from files and the environment.
//
//	logger, closer, err := log.New(cfg.Log)
//	if err != nil {
//		return err
//	}
//	defer closer.Close()
//	slog.SetDefault(logger)
//
// That covers the level, the format, the destination, rotation and sampling.
// Anything past it, a second destination or a filter of its own, is a program
// putting the handlers together itself, which is all New does.
//
// # Cost
//
// Both handlers format into a pooled buffer and write it with one call, so a
// record from one goroutine never lands inside a record from another. The
// attributes a logger carries from [slog.Logger.With] are formatted once, when
// With is called, and copied into each record after that.
//
// A record allocates nothing, including one carrying an error, which is why
// the error is written by appending its parts rather than by building a string.
// The exception is the stack: resolving program counters into file and line
// allocates, and it happens once, on the records that print one.
package log
