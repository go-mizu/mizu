package cli

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
)

// opKind represents the type of file operation.
type opKind int

const (
	opMkdir opKind = iota
	opWrite
	opSkip
)

func (k opKind) String() string {
	switch k {
	case opMkdir:
		return "mkdir"
	case opWrite:
		return "write"
	case opSkip:
		return "skip"
	default:
		return "unknown"
	}
}

// op represents a single file operation.
type op struct {
	kind    opKind
	path    string // relative to root
	content []byte // for write ops
	mode    fs.FileMode
	reason  string // for skipped ops
}

// plan represents a set of file operations to apply.
type plan struct {
	template string
	root     string
	ops      []op
}

// newPlan creates a new plan for the given template and root.
func newPlan(template, root string) *plan {
	return &plan{
		template: template,
		root:     root,
	}
}

// addMkdir adds a directory creation operation.
func (p *plan) addMkdir(path string) error {
	if err := validatePath(path); err != nil {
		return err
	}
	p.ops = append(p.ops, op{
		kind: opMkdir,
		path: path,
		mode: defaultDirMode,
	})
	return nil
}

// addWrite adds a file write operation.
func (p *plan) addWrite(path string, content []byte, mode fs.FileMode) error {
	if err := validatePath(path); err != nil {
		return err
	}
	if mode == 0 {
		mode = defaultFileMode
	}
	p.ops = append(p.ops, op{
		kind:    opWrite,
		path:    path,
		content: content,
		mode:    mode,
	})
	return nil
}

// addSkip adds a skipped file operation.
func (p *plan) addSkip(path, reason string) {
	p.ops = append(p.ops, op{
		kind:   opSkip,
		path:   path,
		reason: reason,
	})
}

// sort orders operations for deterministic execution.
// Directories before files, then alphabetically.
func (p *plan) sort() {
	sort.Slice(p.ops, func(i, j int) bool {
		// Directories first
		if p.ops[i].kind == opMkdir && p.ops[j].kind != opMkdir {
			return true
		}
		if p.ops[i].kind != opMkdir && p.ops[j].kind == opMkdir {
			return false
		}
		// Then alphabetically
		return p.ops[i].path < p.ops[j].path
	})
}

// summary returns counts of each operation type.
func (p *plan) summary() (mkdir, write, overwrite, skip int) {
	for _, o := range p.ops {
		switch o.kind {
		case opMkdir:
			mkdir++
		case opWrite:
			// Check if file exists
			fullPath := filepath.Join(p.root, o.path)
			if fileExists(fullPath) {
				overwrite++
			} else {
				write++
			}
		case opSkip:
			skip++
		}
	}
	return
}

// checkConflicts returns paths that would be overwritten.
func (p *plan) checkConflicts() []string {
	var conflicts []string
	for _, o := range p.ops {
		if o.kind != opWrite {
			continue
		}
		fullPath := filepath.Join(p.root, o.path)
		if fileExists(fullPath) {
			conflicts = append(conflicts, o.path)
		}
	}
	return conflicts
}

// apply executes all operations in the plan.
func (p *plan) apply(force bool) error {
	// Check conflicts first if not forcing
	if !force {
		conflicts := p.checkConflicts()
		if len(conflicts) > 0 {
			return fmt.Errorf("files already exist: %v (use --force to overwrite)", conflicts)
		}
	}

	for _, o := range p.ops {
		if o.kind == opSkip {
			continue
		}

		fullPath := filepath.Join(p.root, o.path)

		switch o.kind {
		case opMkdir:
			if err := ensureDir(fullPath); err != nil {
				return fmt.Errorf("mkdir %s: %w", o.path, err)
			}
		case opWrite:
			if err := atomicWrite(fullPath, o.content, o.mode); err != nil {
				return fmt.Errorf("write %s: %w", o.path, err)
			}
		}
	}

	return nil
}

// printHuman outputs the plan in human-readable format.
func (p *plan) printHuman(out *output) {
	out.print("Plan: create %s (template: %s)\n\n", out.cyan(p.root), out.bold(p.template))

	for _, o := range p.ops {
		switch o.kind {
		case opMkdir:
			out.print("%s  %s\n", out.green("+ mkdir "), o.path)
		case opWrite:
			fullPath := filepath.Join(p.root, o.path)
			if fileExists(fullPath) {
				out.print("%s  %s %s\n", out.yellow("~ write "), o.path, out.gray("(overwrite)"))
			} else {
				out.print("%s  %s\n", out.green("+ write "), o.path)
			}
		case opSkip:
			out.print("%s  %s %s\n", out.gray("- skip  "), o.path, out.gray("("+o.reason+")"))
		}
	}

	mkdir, write, overwrite, skip := p.summary()
	out.print("\nSummary:\n")
	out.print("  create dirs: %d\n", mkdir)
	out.print("  write files: %d\n", write)
	out.print("  overwrite: %d\n", overwrite)
	out.print("  skipped: %d\n", skip)
}

// planJSON is the JSON representation of a plan.
type planJSON struct {
	Template string      `json:"template"`
	Root     string      `json:"root"`
	Ops      []opJSON    `json:"ops"`
	Summary  summaryJSON `json:"summary"`
}

type opJSON struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Bytes int    `json:"bytes,omitempty"`
	Mode  string `json:"mode,omitempty"`
}

type summaryJSON struct {
	Mkdir     int `json:"mkdir"`
	Write     int `json:"write"`
	Overwrite int `json:"overwrite"`
	Skip      int `json:"skip"`
}

// toJSON converts the plan to JSON format.
func (p *plan) toJSON() planJSON {
	mkdir, write, overwrite, skip := p.summary()

	var ops []opJSON
	for _, o := range p.ops {
		oj := opJSON{
			Op:   o.kind.String(),
			Path: o.path,
		}
		if o.kind == opWrite {
			oj.Bytes = len(o.content)
			oj.Mode = fmt.Sprintf("%04o", o.mode)
		}
		ops = append(ops, oj)
	}

	return planJSON{
		Template: p.template,
		Root:     p.root,
		Ops:      ops,
		Summary: summaryJSON{
			Mkdir:     mkdir,
			Write:     write,
			Overwrite: overwrite,
			Skip:      skip,
		},
	}
}
