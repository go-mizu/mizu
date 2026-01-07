package formula

import (
	"regexp"
	"strconv"
	"strings"
)

// CellRef represents a parsed cell reference.
type CellRef struct {
	Sheet    string // Sheet name (empty for same sheet)
	Col      int    // 0-indexed column
	Row      int    // 0-indexed row
	ColAbs   bool   // Column is absolute ($A)
	RowAbs   bool   // Row is absolute ($1)
	Original string // Original reference string
}

// RangeRef represents a parsed range reference.
type RangeRef struct {
	Sheet    string   // Sheet name (empty for same sheet)
	Start    *CellRef // Start cell
	End      *CellRef // End cell
	Original string   // Original reference string
}

// cellRefRegex matches cell references like A1, $A$1, Sheet1!A1, 'Sheet Name'!A1
var cellRefRegex = regexp.MustCompile(`(?:'([^']+)'!|([A-Za-z][A-Za-z0-9_]*!))?(\$)?([A-Za-z]+)(\$)?(\d+)`)

// rangeRegex matches range references like A1:B10, $A$1:$B$10
var rangeRegex = regexp.MustCompile(`(?:'([^']+)'!|([A-Za-z][A-Za-z0-9_]*!))?(\$)?([A-Za-z]+)(\$)?(\d+):(\$)?([A-Za-z]+)(\$)?(\d+)`)

// ShiftFormula adjusts cell references in a formula when rows/columns are inserted or deleted.
// Parameters:
//   - formula: the original formula string
//   - shiftType: "row" or "col"
//   - startIndex: the row/column index where insertion/deletion starts (0-indexed)
//   - count: positive for insert, negative for delete
//   - currentSheet: the current sheet name (for cross-sheet reference handling)
//
// Returns: adjusted formula string
func ShiftFormula(formula string, shiftType string, startIndex, count int, currentSheet string) string {
	if formula == "" {
		return formula
	}

	// Handle ranges first (before cell references to avoid partial matches)
	result := rangeRegex.ReplaceAllStringFunc(formula, func(match string) string {
		ref := ParseRangeReference(match)
		if ref == nil {
			return match
		}

		// Only shift references to the current sheet (empty sheet means same sheet)
		if ref.Sheet != "" && ref.Sheet != currentSheet {
			return match
		}

		shifted := shiftRangeRef(ref, shiftType, startIndex, count)
		return BuildRangeReference(shifted)
	})

	// Then handle single cell references
	// But we need to avoid re-matching parts of ranges that we already processed
	// Use a different approach: find all cell refs that are NOT part of ranges
	result = replaceCellRefsNotInRanges(result, func(match string) string {
		ref := ParseCellReference(match)
		if ref == nil {
			return match
		}

		// Only shift references to the current sheet
		if ref.Sheet != "" && ref.Sheet != currentSheet {
			return match
		}

		shifted := shiftCellRef(ref, shiftType, startIndex, count)
		return BuildCellReference(shifted)
	})

	return result
}

// replaceCellRefsNotInRanges replaces cell references that are not part of range references.
func replaceCellRefsNotInRanges(formula string, replacer func(string) string) string {
	// Find positions of all range references to exclude
	rangeMatches := rangeRegex.FindAllStringIndex(formula, -1)
	excludeRanges := make(map[int]bool)
	for _, match := range rangeMatches {
		for i := match[0]; i < match[1]; i++ {
			excludeRanges[i] = true
		}
	}

	// Build result by processing cell references
	result := strings.Builder{}
	lastEnd := 0

	cellMatches := cellRefRegex.FindAllStringSubmatchIndex(formula, -1)
	for _, match := range cellMatches {
		start, end := match[0], match[1]

		// Skip if this is part of a range
		if excludeRanges[start] {
			continue
		}

		// Check if followed by : (would make it a range start) - skip since ranges are handled separately
		if end < len(formula) && formula[end] == ':' {
			continue
		}

		// Check if preceded by : (would make it a range end)
		if start > 0 && formula[start-1] == ':' {
			continue
		}

		// Write everything before this match
		result.WriteString(formula[lastEnd:start])

		// Replace the match
		originalRef := formula[start:end]
		result.WriteString(replacer(originalRef))

		lastEnd = end
	}

	// Write remaining
	result.WriteString(formula[lastEnd:])

	return result.String()
}

// ParseCellReference parses a cell reference string into a CellRef struct.
func ParseCellReference(ref string) *CellRef {
	matches := cellRefRegex.FindStringSubmatch(ref)
	if matches == nil {
		return nil
	}

	result := &CellRef{Original: ref}

	// Sheet name (quoted or unquoted)
	if matches[1] != "" {
		result.Sheet = matches[1]
	} else if matches[2] != "" {
		result.Sheet = strings.TrimSuffix(matches[2], "!")
	}

	// Column absolute marker
	result.ColAbs = matches[3] == "$"

	// Column letter(s)
	colStr := strings.ToUpper(matches[4])
	col := 0
	for _, c := range colStr {
		col = col*26 + int(c-'A') + 1
	}
	result.Col = col - 1 // 0-indexed

	// Row absolute marker
	result.RowAbs = matches[5] == "$"

	// Row number
	row, _ := strconv.Atoi(matches[6])
	result.Row = row - 1 // 0-indexed

	return result
}

// ParseRangeReference parses a range reference string into a RangeRef struct.
func ParseRangeReference(ref string) *RangeRef {
	matches := rangeRegex.FindStringSubmatch(ref)
	if matches == nil {
		return nil
	}

	result := &RangeRef{Original: ref}

	// Sheet name (quoted or unquoted)
	if matches[1] != "" {
		result.Sheet = matches[1]
	} else if matches[2] != "" {
		result.Sheet = strings.TrimSuffix(matches[2], "!")
	}

	// Start cell
	result.Start = &CellRef{
		Sheet:  result.Sheet,
		ColAbs: matches[3] == "$",
		RowAbs: matches[5] == "$",
	}
	colStr := strings.ToUpper(matches[4])
	col := 0
	for _, c := range colStr {
		col = col*26 + int(c-'A') + 1
	}
	result.Start.Col = col - 1
	row, _ := strconv.Atoi(matches[6])
	result.Start.Row = row - 1

	// End cell
	result.End = &CellRef{
		Sheet:  result.Sheet,
		ColAbs: matches[7] == "$",
		RowAbs: matches[9] == "$",
	}
	colStr = strings.ToUpper(matches[8])
	col = 0
	for _, c := range colStr {
		col = col*26 + int(c-'A') + 1
	}
	result.End.Col = col - 1
	row, _ = strconv.Atoi(matches[10])
	result.End.Row = row - 1

	return result
}

// BuildCellReference builds a cell reference string from a CellRef struct.
func BuildCellReference(ref *CellRef) string {
	if ref == nil {
		return "#REF!"
	}

	if ref.Row < 0 || ref.Col < 0 {
		return "#REF!"
	}

	var sb strings.Builder

	// Sheet name
	if ref.Sheet != "" {
		if strings.ContainsAny(ref.Sheet, " '-+()") {
			sb.WriteString("'")
			sb.WriteString(ref.Sheet)
			sb.WriteString("'!")
		} else {
			sb.WriteString(ref.Sheet)
			sb.WriteString("!")
		}
	}

	// Column
	if ref.ColAbs {
		sb.WriteString("$")
	}
	sb.WriteString(ColToLetter(ref.Col))

	// Row
	if ref.RowAbs {
		sb.WriteString("$")
	}
	sb.WriteString(strconv.Itoa(ref.Row + 1))

	return sb.String()
}

// BuildRangeReference builds a range reference string from a RangeRef struct.
func BuildRangeReference(ref *RangeRef) string {
	if ref == nil || ref.Start == nil || ref.End == nil {
		return "#REF!"
	}

	if ref.Start.Row < 0 || ref.Start.Col < 0 || ref.End.Row < 0 || ref.End.Col < 0 {
		return "#REF!"
	}

	var sb strings.Builder

	// Sheet name (only once at the start)
	if ref.Sheet != "" {
		if strings.ContainsAny(ref.Sheet, " '-+()") {
			sb.WriteString("'")
			sb.WriteString(ref.Sheet)
			sb.WriteString("'!")
		} else {
			sb.WriteString(ref.Sheet)
			sb.WriteString("!")
		}
	}

	// Start cell (without sheet since it's already added)
	if ref.Start.ColAbs {
		sb.WriteString("$")
	}
	sb.WriteString(ColToLetter(ref.Start.Col))
	if ref.Start.RowAbs {
		sb.WriteString("$")
	}
	sb.WriteString(strconv.Itoa(ref.Start.Row + 1))

	sb.WriteString(":")

	// End cell (without sheet)
	if ref.End.ColAbs {
		sb.WriteString("$")
	}
	sb.WriteString(ColToLetter(ref.End.Col))
	if ref.End.RowAbs {
		sb.WriteString("$")
	}
	sb.WriteString(strconv.Itoa(ref.End.Row + 1))

	return sb.String()
}

// shiftCellRef shifts a cell reference based on row/column insertion or deletion.
func shiftCellRef(ref *CellRef, shiftType string, startIndex, count int) *CellRef {
	if ref == nil {
		return nil
	}

	result := &CellRef{
		Sheet:  ref.Sheet,
		Col:    ref.Col,
		Row:    ref.Row,
		ColAbs: ref.ColAbs,
		RowAbs: ref.RowAbs,
	}

	if shiftType == "row" {
		result.Row = shiftIndex(ref.Row, startIndex, count)
	} else if shiftType == "col" {
		result.Col = shiftIndex(ref.Col, startIndex, count)
	}

	return result
}

// shiftRangeRef shifts a range reference based on row/column insertion or deletion.
func shiftRangeRef(ref *RangeRef, shiftType string, startIndex, count int) *RangeRef {
	if ref == nil {
		return nil
	}

	result := &RangeRef{
		Sheet: ref.Sheet,
		Start: shiftCellRef(ref.Start, shiftType, startIndex, count),
		End:   shiftCellRef(ref.End, shiftType, startIndex, count),
	}

	return result
}

// shiftIndex adjusts an index based on insertion or deletion.
// Returns -1 if the index falls within a deleted range.
func shiftIndex(index, startIndex, count int) int {
	if count > 0 {
		// Insertion
		if index >= startIndex {
			return index + count
		}
		return index
	} else if count < 0 {
		// Deletion
		deleteCount := -count
		endIndex := startIndex + deleteCount - 1

		if index >= startIndex && index <= endIndex {
			// Index is within deleted range
			return -1
		}
		if index > endIndex {
			return index - deleteCount
		}
		return index
	}
	return index
}

// ShiftFormulas adjusts all formulas in a map of cells when rows/columns are inserted or deleted.
// This is used for batch operations.
func ShiftFormulas(formulas map[string]string, shiftType string, startIndex, count int, currentSheet string) map[string]string {
	result := make(map[string]string)
	for key, formula := range formulas {
		result[key] = ShiftFormula(formula, shiftType, startIndex, count, currentSheet)
	}
	return result
}

// AdjustFormulaForCopy adjusts a formula when copying from one cell to another.
// Relative references are adjusted by the row/column offset.
// Absolute references remain unchanged.
func AdjustFormulaForCopy(formula string, rowOffset, colOffset int, currentSheet string) string {
	if formula == "" {
		return formula
	}

	// Handle ranges first
	result := rangeRegex.ReplaceAllStringFunc(formula, func(match string) string {
		ref := ParseRangeReference(match)
		if ref == nil {
			return match
		}

		// Only adjust references to the current sheet
		if ref.Sheet != "" && ref.Sheet != currentSheet {
			return match
		}

		// Adjust start cell
		if !ref.Start.RowAbs {
			ref.Start.Row += rowOffset
		}
		if !ref.Start.ColAbs {
			ref.Start.Col += colOffset
		}

		// Adjust end cell
		if !ref.End.RowAbs {
			ref.End.Row += rowOffset
		}
		if !ref.End.ColAbs {
			ref.End.Col += colOffset
		}

		return BuildRangeReference(ref)
	})

	// Then handle single cell references
	result = replaceCellRefsNotInRanges(result, func(match string) string {
		ref := ParseCellReference(match)
		if ref == nil {
			return match
		}

		// Only adjust references to the current sheet
		if ref.Sheet != "" && ref.Sheet != currentSheet {
			return match
		}

		// Adjust based on absolute markers
		if !ref.RowAbs {
			ref.Row += rowOffset
		}
		if !ref.ColAbs {
			ref.Col += colOffset
		}

		return BuildCellReference(ref)
	})

	return result
}

// UpdateSheetReferences updates all references to a sheet when it's renamed.
func UpdateSheetReferences(formula string, oldName, newName string) string {
	if formula == "" {
		return formula
	}

	// Replace quoted references
	oldQuoted := "'" + oldName + "'!"
	newQuoted := "'" + newName + "'!"
	result := strings.ReplaceAll(formula, oldQuoted, newQuoted)

	// Replace unquoted references (only if sheet name is a simple identifier)
	if !strings.ContainsAny(oldName, " '-+()") {
		oldUnquoted := oldName + "!"
		newUnquoted := newName + "!"
		// Need to be careful not to replace partial matches
		// Use word boundary approach
		result = replaceSheetName(result, oldUnquoted, newUnquoted)
	}

	return result
}

// replaceSheetName replaces sheet references while avoiding partial matches.
func replaceSheetName(formula, oldRef, newRef string) string {
	var sb strings.Builder
	i := 0
	for i < len(formula) {
		// Check if we're at a potential sheet reference
		if strings.HasPrefix(formula[i:], oldRef) {
			// Make sure it's not part of a larger identifier
			// (check that previous char is not alphanumeric or _)
			if i > 0 {
				prevChar := formula[i-1]
				if (prevChar >= 'A' && prevChar <= 'Z') ||
					(prevChar >= 'a' && prevChar <= 'z') ||
					(prevChar >= '0' && prevChar <= '9') ||
					prevChar == '_' {
					sb.WriteByte(formula[i])
					i++
					continue
				}
			}
			sb.WriteString(newRef)
			i += len(oldRef)
		} else {
			sb.WriteByte(formula[i])
			i++
		}
	}
	return sb.String()
}

// MarkDeletedSheetReferences replaces all references to a deleted sheet with #REF!.
func MarkDeletedSheetReferences(formula string, deletedSheet string) string {
	if formula == "" {
		return formula
	}

	// Handle quoted sheet name
	quotedPattern := regexp.MustCompile(`'` + regexp.QuoteMeta(deletedSheet) + `'!\$?[A-Za-z]+\$?\d+(:\$?[A-Za-z]+\$?\d+)?`)
	result := quotedPattern.ReplaceAllString(formula, "#REF!")

	// Handle unquoted sheet name
	if !strings.ContainsAny(deletedSheet, " '-+()") {
		unquotedPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(deletedSheet) + `!\$?[A-Za-z]+\$?\d+(:\$?[A-Za-z]+\$?\d+)?`)
		result = unquotedPattern.ReplaceAllString(result, "#REF!")
	}

	return result
}
