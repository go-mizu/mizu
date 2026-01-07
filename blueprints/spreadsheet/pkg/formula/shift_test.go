package formula

import (
	"testing"
)

func TestParseCellReference(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *CellRef
	}{
		{
			name:  "simple reference",
			input: "A1",
			expected: &CellRef{
				Col: 0, Row: 0,
				ColAbs: false, RowAbs: false,
			},
		},
		{
			name:  "absolute reference",
			input: "$A$1",
			expected: &CellRef{
				Col: 0, Row: 0,
				ColAbs: true, RowAbs: true,
			},
		},
		{
			name:  "mixed reference col abs",
			input: "$A1",
			expected: &CellRef{
				Col: 0, Row: 0,
				ColAbs: true, RowAbs: false,
			},
		},
		{
			name:  "mixed reference row abs",
			input: "A$1",
			expected: &CellRef{
				Col: 0, Row: 0,
				ColAbs: false, RowAbs: true,
			},
		},
		{
			name:  "multi-letter column",
			input: "AA10",
			expected: &CellRef{
				Col: 26, Row: 9,
				ColAbs: false, RowAbs: false,
			},
		},
		{
			name:  "sheet reference unquoted",
			input: "Sheet1!B2",
			expected: &CellRef{
				Sheet: "Sheet1",
				Col:   1, Row: 1,
				ColAbs: false, RowAbs: false,
			},
		},
		{
			name:  "sheet reference quoted",
			input: "'My Sheet'!C3",
			expected: &CellRef{
				Sheet: "My Sheet",
				Col:   2, Row: 2,
				ColAbs: false, RowAbs: false,
			},
		},
		{
			name:  "sheet reference with absolute",
			input: "Sheet1!$D$4",
			expected: &CellRef{
				Sheet: "Sheet1",
				Col:   3, Row: 3,
				ColAbs: true, RowAbs: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCellReference(tt.input)
			if result == nil {
				t.Fatalf("ParseCellReference returned nil for %q", tt.input)
			}
			if result.Col != tt.expected.Col {
				t.Errorf("Col: got %d, want %d", result.Col, tt.expected.Col)
			}
			if result.Row != tt.expected.Row {
				t.Errorf("Row: got %d, want %d", result.Row, tt.expected.Row)
			}
			if result.ColAbs != tt.expected.ColAbs {
				t.Errorf("ColAbs: got %v, want %v", result.ColAbs, tt.expected.ColAbs)
			}
			if result.RowAbs != tt.expected.RowAbs {
				t.Errorf("RowAbs: got %v, want %v", result.RowAbs, tt.expected.RowAbs)
			}
			if result.Sheet != tt.expected.Sheet {
				t.Errorf("Sheet: got %q, want %q", result.Sheet, tt.expected.Sheet)
			}
		})
	}
}

func TestParseRangeReference(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		startCol     int
		startRow     int
		endCol       int
		endRow       int
		sheet        string
		startColAbs  bool
		startRowAbs  bool
		endColAbs    bool
		endRowAbs    bool
	}{
		{
			name:     "simple range",
			input:    "A1:B10",
			startCol: 0, startRow: 0,
			endCol: 1, endRow: 9,
		},
		{
			name:     "absolute range",
			input:    "$A$1:$B$10",
			startCol: 0, startRow: 0,
			endCol: 1, endRow: 9,
			startColAbs: true, startRowAbs: true,
			endColAbs: true, endRowAbs: true,
		},
		{
			name:     "mixed range",
			input:    "$A1:B$10",
			startCol: 0, startRow: 0,
			endCol: 1, endRow: 9,
			startColAbs: true, startRowAbs: false,
			endColAbs: false, endRowAbs: true,
		},
		{
			name:     "sheet range",
			input:    "Sheet1!A1:C5",
			sheet:    "Sheet1",
			startCol: 0, startRow: 0,
			endCol: 2, endRow: 4,
		},
		{
			name:     "quoted sheet range",
			input:    "'My Sheet'!D1:F10",
			sheet:    "My Sheet",
			startCol: 3, startRow: 0,
			endCol: 5, endRow: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseRangeReference(tt.input)
			if result == nil {
				t.Fatalf("ParseRangeReference returned nil for %q", tt.input)
			}
			if result.Start.Col != tt.startCol {
				t.Errorf("Start.Col: got %d, want %d", result.Start.Col, tt.startCol)
			}
			if result.Start.Row != tt.startRow {
				t.Errorf("Start.Row: got %d, want %d", result.Start.Row, tt.startRow)
			}
			if result.End.Col != tt.endCol {
				t.Errorf("End.Col: got %d, want %d", result.End.Col, tt.endCol)
			}
			if result.End.Row != tt.endRow {
				t.Errorf("End.Row: got %d, want %d", result.End.Row, tt.endRow)
			}
			if result.Sheet != tt.sheet {
				t.Errorf("Sheet: got %q, want %q", result.Sheet, tt.sheet)
			}
		})
	}
}

func TestBuildCellReference(t *testing.T) {
	tests := []struct {
		name     string
		ref      *CellRef
		expected string
	}{
		{
			name:     "simple",
			ref:      &CellRef{Col: 0, Row: 0},
			expected: "A1",
		},
		{
			name:     "absolute",
			ref:      &CellRef{Col: 0, Row: 0, ColAbs: true, RowAbs: true},
			expected: "$A$1",
		},
		{
			name:     "with sheet",
			ref:      &CellRef{Sheet: "Sheet1", Col: 1, Row: 1},
			expected: "Sheet1!B2",
		},
		{
			name:     "with quoted sheet",
			ref:      &CellRef{Sheet: "My Sheet", Col: 2, Row: 2},
			expected: "'My Sheet'!C3",
		},
		{
			name:     "negative row",
			ref:      &CellRef{Col: 0, Row: -1},
			expected: "#REF!",
		},
		{
			name:     "negative col",
			ref:      &CellRef{Col: -1, Row: 0},
			expected: "#REF!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildCellReference(tt.ref)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestShiftFormula_RowInsert(t *testing.T) {
	tests := []struct {
		name       string
		formula    string
		startIndex int
		count      int
		expected   string
	}{
		{
			name:       "simple ref after insert point",
			formula:    "=A5",
			startIndex: 2, // Insert at row 3 (0-indexed: 2)
			count:      1,
			expected:   "=A6",
		},
		{
			name:       "simple ref before insert point",
			formula:    "=A2",
			startIndex: 2,
			count:      1,
			expected:   "=A2",
		},
		{
			name:       "simple ref at insert point",
			formula:    "=A3",
			startIndex: 2,
			count:      1,
			expected:   "=A4",
		},
		{
			name:       "range spanning insert point",
			formula:    "=SUM(A1:A10)",
			startIndex: 4,
			count:      2,
			expected:   "=SUM(A1:A12)",
		},
		{
			name:       "range after insert point",
			formula:    "=SUM(A5:A10)",
			startIndex: 2,
			count:      1,
			expected:   "=SUM(A6:A11)",
		},
		{
			name:       "range before insert point",
			formula:    "=SUM(A1:A3)",
			startIndex: 4,
			count:      1,
			expected:   "=SUM(A1:A3)",
		},
		{
			name:       "absolute ref after insert",
			formula:    "=$A$5",
			startIndex: 2,
			count:      1,
			expected:   "=$A$6",
		},
		{
			name:       "mixed ref row abs",
			formula:    "=A$5",
			startIndex: 2,
			count:      1,
			expected:   "=A$6",
		},
		{
			name:       "mixed ref col abs",
			formula:    "=$A5",
			startIndex: 2,
			count:      1,
			expected:   "=$A6",
		},
		{
			name:       "multiple refs",
			formula:    "=A5+B5+C5",
			startIndex: 2,
			count:      1,
			expected:   "=A6+B6+C6",
		},
		{
			name:       "function with range and ref",
			formula:    "=SUM(A1:A5)*B5",
			startIndex: 2,
			count:      1,
			expected:   "=SUM(A1:A6)*B6",
		},
		{
			name:       "insert multiple rows",
			formula:    "=A10",
			startIndex: 5,
			count:      3,
			expected:   "=A13",
		},
		{
			name:       "complex formula",
			formula:    "=IF(A5>10,SUM(B1:B5),AVERAGE(C5:C10))",
			startIndex: 2,
			count:      1,
			expected:   "=IF(A6>10,SUM(B1:B6),AVERAGE(C6:C11))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShiftFormula(tt.formula, "row", tt.startIndex, tt.count, "")
			if result != tt.expected {
				t.Errorf("ShiftFormula(%q, row, %d, %d) = %q, want %q",
					tt.formula, tt.startIndex, tt.count, result, tt.expected)
			}
		})
	}
}

func TestShiftFormula_RowDelete(t *testing.T) {
	tests := []struct {
		name       string
		formula    string
		startIndex int
		count      int
		expected   string
	}{
		{
			name:       "ref after deleted rows",
			formula:    "=A10",
			startIndex: 4,
			count:      -2,
			expected:   "=A8",
		},
		{
			name:       "ref in deleted range",
			formula:    "=A5",
			startIndex: 4, // Delete row 5 (0-indexed: 4)
			count:      -1,
			expected:   "=#REF!",
		},
		{
			name:       "ref before deleted range",
			formula:    "=A3",
			startIndex: 4,
			count:      -2,
			expected:   "=A3",
		},
		{
			name:       "range with deleted start",
			formula:    "=SUM(A5:A10)",
			startIndex: 4,
			count:      -1,
			expected:   "=SUM(#REF!)", // When range start is deleted, entire range becomes #REF!
		},
		{
			name:       "range fully deleted",
			formula:    "=SUM(A5:A6)",
			startIndex: 4,
			count:      -2,
			expected:   "=SUM(#REF!)",
		},
		{
			name:       "range partially deleted end",
			formula:    "=SUM(A1:A10)",
			startIndex: 4,
			count:      -2,
			expected:   "=SUM(A1:A8)",
		},
		{
			name:       "range before deleted range",
			formula:    "=SUM(A1:A3)",
			startIndex: 4,
			count:      -2,
			expected:   "=SUM(A1:A3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShiftFormula(tt.formula, "row", tt.startIndex, tt.count, "")
			if result != tt.expected {
				t.Errorf("ShiftFormula(%q, row, %d, %d) = %q, want %q",
					tt.formula, tt.startIndex, tt.count, result, tt.expected)
			}
		})
	}
}

func TestShiftFormula_ColInsert(t *testing.T) {
	tests := []struct {
		name       string
		formula    string
		startIndex int
		count      int
		expected   string
	}{
		{
			name:       "simple ref after insert",
			formula:    "=E1",
			startIndex: 1, // Insert at column B (0-indexed: 1)
			count:      1,
			expected:   "=F1",
		},
		{
			name:       "simple ref before insert",
			formula:    "=A1",
			startIndex: 1,
			count:      1,
			expected:   "=A1",
		},
		{
			name:       "range spanning insert",
			formula:    "=SUM(A1:E1)",
			startIndex: 1,
			count:      2,
			expected:   "=SUM(A1:G1)",
		},
		{
			name:       "range after insert",
			formula:    "=SUM(C1:E1)",
			startIndex: 1,
			count:      1,
			expected:   "=SUM(D1:F1)",
		},
		{
			name:       "absolute col ref",
			formula:    "=$E$1",
			startIndex: 1,
			count:      1,
			expected:   "=$F$1",
		},
		{
			name:       "mixed col abs",
			formula:    "=$E1",
			startIndex: 1,
			count:      1,
			expected:   "=$F1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShiftFormula(tt.formula, "col", tt.startIndex, tt.count, "")
			if result != tt.expected {
				t.Errorf("ShiftFormula(%q, col, %d, %d) = %q, want %q",
					tt.formula, tt.startIndex, tt.count, result, tt.expected)
			}
		})
	}
}

func TestShiftFormula_ColDelete(t *testing.T) {
	tests := []struct {
		name       string
		formula    string
		startIndex int
		count      int
		expected   string
	}{
		{
			name:       "ref after deleted cols",
			formula:    "=E1",
			startIndex: 1, // Delete column B (0-indexed: 1)
			count:      -2,
			expected:   "=C1",
		},
		{
			name:       "ref in deleted range",
			formula:    "=B1",
			startIndex: 1,
			count:      -1,
			expected:   "=#REF!",
		},
		{
			name:       "ref before deleted range",
			formula:    "=A1",
			startIndex: 1,
			count:      -2,
			expected:   "=A1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShiftFormula(tt.formula, "col", tt.startIndex, tt.count, "")
			if result != tt.expected {
				t.Errorf("ShiftFormula(%q, col, %d, %d) = %q, want %q",
					tt.formula, tt.startIndex, tt.count, result, tt.expected)
			}
		})
	}
}

func TestShiftFormula_CrossSheet(t *testing.T) {
	tests := []struct {
		name         string
		formula      string
		shiftType    string
		startIndex   int
		count        int
		currentSheet string
		expected     string
	}{
		{
			name:         "same sheet ref shifted",
			formula:      "=A5",
			shiftType:    "row",
			startIndex:   2,
			count:        1,
			currentSheet: "",
			expected:     "=A6",
		},
		{
			name:         "cross-sheet ref not shifted",
			formula:      "=Sheet1!A5",
			shiftType:    "row",
			startIndex:   2,
			count:        1,
			currentSheet: "Sheet2",
			expected:     "=Sheet1!A5",
		},
		{
			name:         "cross-sheet ref shifted when matching",
			formula:      "=Sheet1!A5",
			shiftType:    "row",
			startIndex:   2,
			count:        1,
			currentSheet: "Sheet1",
			expected:     "=Sheet1!A6",
		},
		{
			name:         "mixed refs different sheets",
			formula:      "=A5+Sheet1!A5",
			shiftType:    "row",
			startIndex:   2,
			count:        1,
			currentSheet: "Sheet2",
			expected:     "=A6+Sheet1!A5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShiftFormula(tt.formula, tt.shiftType, tt.startIndex, tt.count, tt.currentSheet)
			if result != tt.expected {
				t.Errorf("ShiftFormula(%q, %s, %d, %d, %q) = %q, want %q",
					tt.formula, tt.shiftType, tt.startIndex, tt.count, tt.currentSheet, result, tt.expected)
			}
		})
	}
}

func TestAdjustFormulaForCopy(t *testing.T) {
	tests := []struct {
		name      string
		formula   string
		rowOffset int
		colOffset int
		expected  string
	}{
		{
			name:      "copy right",
			formula:   "=A1",
			rowOffset: 0,
			colOffset: 1,
			expected:  "=B1",
		},
		{
			name:      "copy down",
			formula:   "=A1",
			rowOffset: 1,
			colOffset: 0,
			expected:  "=A2",
		},
		{
			name:      "copy diagonal",
			formula:   "=A1",
			rowOffset: 2,
			colOffset: 2,
			expected:  "=C3",
		},
		{
			name:      "absolute not changed",
			formula:   "=$A$1",
			rowOffset: 1,
			colOffset: 1,
			expected:  "=$A$1",
		},
		{
			name:      "mixed row abs",
			formula:   "=A$1",
			rowOffset: 1,
			colOffset: 1,
			expected:  "=B$1",
		},
		{
			name:      "mixed col abs",
			formula:   "=$A1",
			rowOffset: 1,
			colOffset: 1,
			expected:  "=$A2",
		},
		{
			name:      "range copy",
			formula:   "=SUM(A1:B5)",
			rowOffset: 1,
			colOffset: 1,
			expected:  "=SUM(B2:C6)",
		},
		{
			name:      "range with absolute",
			formula:   "=SUM($A$1:B5)",
			rowOffset: 1,
			colOffset: 1,
			expected:  "=SUM($A$1:C6)",
		},
		{
			name:      "complex formula",
			formula:   "=A1+$B$2*C3",
			rowOffset: 1,
			colOffset: 1,
			expected:  "=B2+$B$2*D4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AdjustFormulaForCopy(tt.formula, tt.rowOffset, tt.colOffset, "")
			if result != tt.expected {
				t.Errorf("AdjustFormulaForCopy(%q, %d, %d) = %q, want %q",
					tt.formula, tt.rowOffset, tt.colOffset, result, tt.expected)
			}
		})
	}
}

func TestUpdateSheetReferences(t *testing.T) {
	tests := []struct {
		name     string
		formula  string
		oldName  string
		newName  string
		expected string
	}{
		{
			name:     "unquoted sheet name",
			formula:  "=Sheet1!A1",
			oldName:  "Sheet1",
			newName:  "Data",
			expected: "=Data!A1",
		},
		{
			name:     "quoted sheet name",
			formula:  "='My Sheet'!A1",
			oldName:  "My Sheet",
			newName:  "New Name",
			expected: "='New Name'!A1",
		},
		{
			name:     "multiple references",
			formula:  "=Sheet1!A1+Sheet1!B1",
			oldName:  "Sheet1",
			newName:  "Data",
			expected: "=Data!A1+Data!B1",
		},
		{
			name:     "range reference",
			formula:  "=SUM(Sheet1!A1:B10)",
			oldName:  "Sheet1",
			newName:  "Data",
			expected: "=SUM(Data!A1:B10)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UpdateSheetReferences(tt.formula, tt.oldName, tt.newName)
			if result != tt.expected {
				t.Errorf("UpdateSheetReferences(%q, %q, %q) = %q, want %q",
					tt.formula, tt.oldName, tt.newName, result, tt.expected)
			}
		})
	}
}

func TestMarkDeletedSheetReferences(t *testing.T) {
	tests := []struct {
		name         string
		formula      string
		deletedSheet string
		expected     string
	}{
		{
			name:         "simple reference",
			formula:      "=Sheet1!A1",
			deletedSheet: "Sheet1",
			expected:     "=#REF!",
		},
		{
			name:         "mixed with local ref",
			formula:      "=A1+Sheet1!B1",
			deletedSheet: "Sheet1",
			expected:     "=A1+#REF!",
		},
		{
			name:         "range reference",
			formula:      "=SUM(Sheet1!A1:B10)",
			deletedSheet: "Sheet1",
			expected:     "=SUM(#REF!)",
		},
		{
			name:         "different sheet not affected",
			formula:      "=Sheet2!A1",
			deletedSheet: "Sheet1",
			expected:     "=Sheet2!A1",
		},
		{
			name:         "quoted sheet name",
			formula:      "='My Sheet'!A1",
			deletedSheet: "My Sheet",
			expected:     "=#REF!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MarkDeletedSheetReferences(tt.formula, tt.deletedSheet)
			if result != tt.expected {
				t.Errorf("MarkDeletedSheetReferences(%q, %q) = %q, want %q",
					tt.formula, tt.deletedSheet, result, tt.expected)
			}
		})
	}
}

func TestShiftIndex(t *testing.T) {
	tests := []struct {
		name       string
		index      int
		startIndex int
		count      int
		expected   int
	}{
		// Insertion tests
		{"insert: index before", 1, 5, 2, 1},
		{"insert: index at start", 5, 5, 2, 7},
		{"insert: index after start", 7, 5, 2, 9},

		// Deletion tests
		{"delete: index before", 2, 5, -2, 2},
		{"delete: index in range start", 5, 5, -2, -1},
		{"delete: index in range middle", 6, 5, -2, -1},
		{"delete: index after range", 8, 5, -2, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shiftIndex(tt.index, tt.startIndex, tt.count)
			if result != tt.expected {
				t.Errorf("shiftIndex(%d, %d, %d) = %d, want %d",
					tt.index, tt.startIndex, tt.count, result, tt.expected)
			}
		})
	}
}
