# Spec 0047: Modern CLI Output Formatting

## Status
**Complete**

## Overview
Consolidate and modernize the Mizu CLI output formatting by removing legacy code, standardizing on Fang/Lipgloss styling, and ensuring consistent formatting across all commands.

## Current State Analysis

### Fang Integration (Already Complete)
Fang is already integrated in `cmd/cli/root.go`:
- `fang.Execute()` wraps Cobra with styled help
- Automatic version handling via `fang.WithVersion()` and `fang.WithCommit()`
- Man pages and shell completions provided automatically
- Global flags: `--json`, `--no-color`, `--quiet`, `--verbose`, `--md`

### Two Output Systems (Problem)
The CLI currently has **two competing output systems**:

1. **Modern Output** (`output.go` - 125 lines):
   - `Output` struct with exported methods: `Print()`, `Errorf()`, `Verbosef()`, `WriteJSON()`, `WriteJSONError()`
   - Color methods: `Title()`, `Bold()`, `Cyan()`, `Dim()`, `Success()`, `Warn()`
   - Uses **Lipgloss** for styling (blue=12, red=9, green=10, yellow=11, gray=8, cyan=14)
   - Respects `--no-color` flag and `NO_COLOR` env var
   - Used by: version, new, dev, contract, middleware commands

2. **Legacy Output** (`compat.go` - 79 lines):
   - `output` struct with lowercase methods: `print()`, `errorf()`, `verbosef()`
   - Uses raw ANSI codes (`\033[32m`, etc.)
   - Only used by: `plan.go` for `--dry-run` output

### Formatting Inconsistencies

#### 1. Error Message Formats
Different patterns used across commands:
```go
// Pattern 1: "Error:" prefix
out.Errorf("Error: %v\n", err)

// Pattern 2: "Error:" with context
out.Errorf("Error: unknown template %q\n", newFlags.template)

// Pattern 3: Plain error (no prefix)
out.Errorf("method not found: %s\n", methodName)
```

#### 2. Hint Message Formats
Inconsistent hint/help messaging:
```go
// Pattern 1: "hint:" prefix
out.Print("\nhint: is the server running? try: mizu dev\n")

// Pattern 2: Plain suggestion
out.Print("Run 'mizu new --list' to see available templates.\n")

// Pattern 3: Inline hints
out.Print("\nhint: mizu contract call %s '{...}'\n", method.FullName)
```

#### 3. Direct fmt.Print Usage
`contract.go` bypasses Output for raw responses:
```go
// Lines 391, 395, 397, 452 use fmt.Print directly
fmt.Print(string(result))
fmt.Println(buf.String())
fmt.Print(string(specData))
```
This is **intentional** for raw output (spec content, method results) but should be documented.

### Files Inventory

| File | Lines | Purpose | Output System |
|------|-------|---------|---------------|
| root.go | 153 | Fang setup, global flags | Modern |
| output.go | 125 | Modern styled output | N/A (defines it) |
| compat.go | 79 | Legacy ANSI output | N/A (defines it) |
| new.go | 275 | Project scaffolding | Modern |
| dev.go | 218 | Development runner | Modern |
| contract.go | 639 | Contract commands | Modern + fmt.Print |
| middleware.go | 229 | Middleware explorer | Modern |
| version.go | 37 | Version info | Modern |
| plan.go | 258 | Template execution plan | **Legacy** |
| format.go | 74 | Table formatting | Generic |

## Goals

1. **Remove Legacy Output**: Delete `compat.go` and migrate `plan.go` to use modern `Output`
2. **Standardize Error Format**: Consistent error message styling
3. **Standardize Hint Format**: Consistent hint/suggestion styling
4. **Document Intent**: Make raw fmt.Print usage explicit and intentional
5. **Add Missing Styles**: Error prefix style, hint style for consistency

## Implementation Plan

### Phase 1: Add Missing Style Helpers

Add new styling methods to `output.go`:

```go
// Error renders an error prefix consistently
func (o *Output) Error(text string) string {
    if o.noColor {
        return "Error: " + text
    }
    return errorStyle.Render("Error:") + " " + text
}

// Hint renders a hint prefix consistently
func (o *Output) Hint(text string) string {
    if o.noColor {
        return "hint: " + text
    }
    return dimStyle.Render("hint:") + " " + text
}

// Green renders text in green (for success operations)
func (o *Output) Green(text string) string {
    if o.noColor {
        return text
    }
    return successStyle.Render(text)
}

// Yellow renders text in yellow (for warnings/overwrites)
func (o *Output) Yellow(text string) string {
    if o.noColor {
        return text
    }
    return warnStyle.Render(text)
}
```

### Phase 2: Migrate plan.go to Modern Output

Update `plan.printHuman()` to accept `*Output` instead of `*output`:

```go
// Before
func (p *plan) printHuman(out *output) {
    out.print("Plan: create %s (template: %s)\n\n", out.cyan(p.root), out.bold(p.template))
    // ... uses out.green(), out.yellow(), out.gray()
}

// After
func (p *plan) printHuman(out *Output) {
    out.Print("Plan: create %s (template: %s)\n\n", out.Cyan(p.root), out.Bold(p.template))
    // ... uses out.Green(), out.Yellow(), out.Dim()
}
```

Update caller in `new.go`:
```go
// Before
p.printHuman(newLegacyOutput())

// After
p.printHuman(out)
```

### Phase 3: Remove Legacy Output

Delete `compat.go` entirely after migration is complete:
- Delete `type output struct`
- Delete `newOutput()` function
- Delete `newLegacyOutput()` wrapper in `new.go`
- Delete ANSI color constants
- Delete `padRight()` (move to format.go if still needed)

### Phase 4: Standardize Error Messages

Replace all error messages with consistent pattern:

```go
// Standard pattern
out.Errorf("%s\n", out.Error("template is required"))
out.Print("Run 'mizu new --list' to see available templates.\n")

// Or simpler: add PrintError helper
func (o *Output) PrintError(format string, args ...any) {
    msg := fmt.Sprintf(format, args...)
    o.Errorf("%s\n", o.Error(msg))
}
```

### Phase 5: Standardize Hint Messages

Create consistent hint pattern:

```go
// Standard pattern
out.Print("\n%s\n", out.Hint("is the server running? try: mizu dev"))

// Or simpler: add PrintHint helper
func (o *Output) PrintHint(format string, args ...any) {
    msg := fmt.Sprintf(format, args...)
    o.Print("\n%s\n", o.Hint(msg))
}
```

### Phase 6: Document Raw Output Intent

Add comment in `contract.go` to document intentional raw output:

```go
// Output result
// NOTE: Raw fmt.Print is intentional here for unformatted API responses
// that users may pipe to other tools (jq, etc.)
if contractFlags.raw {
    fmt.Print(string(result))
} else {
    // ...
}
```

## Detailed Changes

### File: output.go

Add new methods:
- `Error(text string) string` - styled error prefix
- `Hint(text string) string` - styled hint prefix
- `Green(text string) string` - success color
- `Yellow(text string) string` - warning color
- `PrintError(format string, args ...any)` - convenience for error lines
- `PrintHint(format string, args ...any)` - convenience for hint lines

### File: plan.go

- Change `printHuman(out *output)` to `printHuman(out *Output)`
- Update color method calls: `green()` -> `Green()`, `yellow()` -> `Yellow()`, etc.
- Update `gray()` -> `Dim()`

### File: new.go

- Remove `newLegacyOutput()` function
- Update `p.printHuman(newLegacyOutput())` to `p.printHuman(out)`
- Standardize error/hint messages

### File: contract.go

- Standardize error messages
- Add documentation comments for intentional raw output
- Standardize hint messages

### File: middleware.go

- Standardize error messages
- Standardize hint messages

### File: dev.go

- Standardize error messages

### File: compat.go

- **DELETE** entirely

### File: format.go

- Keep `padRight()` if still needed, otherwise it will be removed with compat.go

## Color Palette (Lipgloss)

Standardized on these ANSI 256 colors:

| Color | Code | Usage |
|-------|------|-------|
| Blue | 12 | Titles, emphasis |
| Red | 9 | Errors |
| Green | 10 | Success, new files |
| Yellow | 11 | Warnings, overwrites |
| Gray | 8 | Dim/secondary text, hints |
| Cyan | 14 | Identifiers, paths |

## Testing Checklist

- [ ] `mizu new --list` shows styled output
- [ ] `mizu new --template api --dry-run` shows styled plan
- [ ] `mizu new --template bad` shows consistent error format
- [ ] `mizu contract ls` with no server shows styled error + hint
- [ ] `mizu middleware show unknown` shows consistent error format
- [ ] `mizu --no-color version` has no ANSI codes
- [ ] `NO_COLOR=1 mizu version` has no ANSI codes
- [ ] JSON output is unaffected by style changes

## Migration Checklist

### Pre-Migration
- [x] Review all output patterns
- [x] Document current inconsistencies
- [x] Identify all files using legacy output

### Migration Steps
- [x] Add new style methods to output.go (Green, Yellow, Error, Hint)
- [x] Update plan.go to use modern Output
- [x] Update new.go to remove legacy wrapper
- [x] Delete compat.go and json.go (unused legacy code)
- [x] Standardize error messages in new.go
- [x] Standardize error messages in contract.go
- [x] Standardize error messages in middleware.go
- [x] Standardize error messages in dev.go
- [x] Standardize hint messages across all files
- [x] Add documentation for raw output in contract.go
- [x] Move padRight to format.go

### Post-Migration
- [x] Build verification passes
- [x] Update CLAUDE.md with CLI structure notes

## Benefits

1. **Reduced Code**: ~80 lines removed (compat.go)
2. **Consistency**: Single output system, predictable styling
3. **Maintainability**: One place to change colors/styles
4. **User Experience**: Consistent error/hint formatting
5. **Future-Proof**: Lipgloss is actively maintained, ANSI codes are fragile

## Trade-offs

### Advantages
- Cleaner codebase (less duplication)
- Consistent user experience
- Easier to maintain and extend

### Disadvantages
- Migration effort (mostly mechanical changes)
- Brief testing period needed

## CLAUDE.md Updates

Add CLI module documentation:
```markdown
### CLI Architecture

The CLI uses Fang (Charmbracelet) wrapping Cobra for enhanced UX:

- **Fang**: Styled help, automatic version/man/completion commands
- **Lipgloss**: Terminal styling for consistent colors
- **Glamour**: Markdown rendering for `--md` flag

Key files:
- `cmd/cli/root.go` - Fang/Cobra setup, global flags
- `cmd/cli/output.go` - Styled output helper (all commands use this)
- `cmd/cli/*.go` - Individual command implementations

Building:
```bash
# Uses GOWORK=off to avoid workspace issues
make install

# Or run directly
make run ARGS="new --list"
```
```

---

**Author**: Claude Code
**Date**: 2025-12-18
**Version**: 1.0
