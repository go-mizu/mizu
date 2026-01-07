package formula

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// CellGetter retrieves cell values.
type CellGetter interface {
	GetCellValue(ctx context.Context, sheetID string, row, col int) (interface{}, error)
	GetRangeValues(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([][]interface{}, error)
	GetNamedRange(ctx context.Context, name string) (sheetID string, startRow, startCol, endRow, endCol int, err error)
	// ResolveSheetName resolves a sheet name to a sheet ID within a workbook.
	// If the name is already a valid ID, it returns it unchanged.
	ResolveSheetName(ctx context.Context, workbookID, sheetName string) (sheetID string, err error)
}

// EvalContext holds context for formula evaluation.
type EvalContext struct {
	SheetID     string
	WorkbookID  string
	CurrentRow  int
	CurrentCol  int
	CellGetter  CellGetter
	Circular    map[string]bool
}

// Evaluator evaluates formula AST nodes.
type Evaluator struct {
	ctx *EvalContext
}

// NewEvaluator creates a new evaluator.
func NewEvaluator(ctx *EvalContext) *Evaluator {
	return &Evaluator{ctx: ctx}
}

// Evaluate evaluates an AST node and returns the result.
func (e *Evaluator) Evaluate(ctx context.Context, node *ASTNode) (interface{}, error) {
	if node == nil {
		return nil, nil
	}

	switch node.Type {
	case NodeNumber:
		return strconv.ParseFloat(node.Value.(string), 64)

	case NodeString:
		return node.Value.(string), nil

	case NodeBool:
		return strings.ToUpper(node.Value.(string)) == "TRUE", nil

	case NodeError:
		return nil, fmt.Errorf("error value: %s", node.Value.(string))

	case NodeReference:
		return e.evaluateReference(ctx, node)

	case NodeRange:
		return e.evaluateRange(ctx, node)

	case NodeName:
		return e.evaluateNamedRange(ctx, node)

	case NodeFunction:
		return e.evaluateFunction(ctx, node)

	case NodeBinaryOp:
		return e.evaluateBinaryOp(ctx, node)

	case NodeUnaryOp:
		return e.evaluateUnaryOp(ctx, node)

	case NodeArray:
		return e.evaluateArray(ctx, node)

	default:
		return nil, fmt.Errorf("unknown node type: %v", node.Type)
	}
}

func (e *Evaluator) evaluateReference(ctx context.Context, node *ASTNode) (interface{}, error) {
	ref := node.Value.(string)
	sheetID := e.ctx.SheetID

	// Check for sheet prefix
	if idx := strings.Index(ref, "!"); idx > 0 {
		sheetName := ref[:idx]
		ref = ref[idx+1:]

		// Strip quotes from sheet name if present
		sheetName = strings.Trim(sheetName, "'")

		// Resolve sheet name to ID
		if e.ctx.CellGetter != nil && e.ctx.WorkbookID != "" {
			resolvedID, err := e.ctx.CellGetter.ResolveSheetName(ctx, e.ctx.WorkbookID, sheetName)
			if err == nil && resolvedID != "" {
				sheetID = resolvedID
			} else {
				// If resolution fails, use the name as-is (might be an ID already)
				sheetID = sheetName
			}
		} else {
			sheetID = sheetName
		}
	}

	row, col, err := ParseCellRef(ref)
	if err != nil {
		return nil, err
	}

	if e.ctx.CellGetter == nil {
		return nil, fmt.Errorf("cell getter not available")
	}

	return e.ctx.CellGetter.GetCellValue(ctx, sheetID, row, col)
}

func (e *Evaluator) evaluateRange(ctx context.Context, node *ASTNode) (interface{}, error) {
	rangeStr := node.Value.(string)
	sheetID := e.ctx.SheetID

	// Check for sheet prefix
	if idx := strings.Index(rangeStr, "!"); idx > 0 {
		sheetName := rangeStr[:idx]
		rangeStr = rangeStr[idx+1:]

		// Strip quotes from sheet name if present
		sheetName = strings.Trim(sheetName, "'")

		// Resolve sheet name to ID
		if e.ctx.CellGetter != nil && e.ctx.WorkbookID != "" {
			resolvedID, err := e.ctx.CellGetter.ResolveSheetName(ctx, e.ctx.WorkbookID, sheetName)
			if err == nil && resolvedID != "" {
				sheetID = resolvedID
			} else {
				// If resolution fails, use the name as-is (might be an ID already)
				sheetID = sheetName
			}
		} else {
			sheetID = sheetName
		}
	}

	startRow, startCol, endRow, endCol, err := ParseRangeRef(rangeStr)
	if err != nil {
		return nil, err
	}

	if e.ctx.CellGetter == nil {
		return nil, fmt.Errorf("cell getter not available")
	}

	return e.ctx.CellGetter.GetRangeValues(ctx, sheetID, startRow, startCol, endRow, endCol)
}

func (e *Evaluator) evaluateNamedRange(ctx context.Context, node *ASTNode) (interface{}, error) {
	name := node.Value.(string)

	if e.ctx.CellGetter == nil {
		return nil, fmt.Errorf("cell getter not available")
	}

	sheetID, startRow, startCol, endRow, endCol, err := e.ctx.CellGetter.GetNamedRange(ctx, name)
	if err != nil {
		return nil, err
	}

	if startRow == endRow && startCol == endCol {
		return e.ctx.CellGetter.GetCellValue(ctx, sheetID, startRow, startCol)
	}

	return e.ctx.CellGetter.GetRangeValues(ctx, sheetID, startRow, startCol, endRow, endCol)
}

func (e *Evaluator) evaluateFunction(ctx context.Context, node *ASTNode) (interface{}, error) {
	funcName := strings.ToUpper(node.Value.(string))
	args := node.Children

	// Get function implementation
	fn, ok := Functions[funcName]
	if !ok {
		return nil, fmt.Errorf("unknown function: %s", funcName)
	}

	// Evaluate arguments
	evaluatedArgs := make([]interface{}, len(args))
	for i, arg := range args {
		val, err := e.Evaluate(ctx, arg)
		if err != nil {
			return nil, fmt.Errorf("error evaluating argument %d of %s: %w", i+1, funcName, err)
		}
		evaluatedArgs[i] = val
	}

	return fn(evaluatedArgs...)
}

func (e *Evaluator) evaluateBinaryOp(ctx context.Context, node *ASTNode) (interface{}, error) {
	if len(node.Children) != 2 {
		return nil, fmt.Errorf("binary operator requires 2 operands")
	}

	left, err := e.Evaluate(ctx, node.Children[0])
	if err != nil {
		return nil, err
	}

	right, err := e.Evaluate(ctx, node.Children[1])
	if err != nil {
		return nil, err
	}

	op := node.Value.(string)

	switch op {
	case "+":
		return toFloat(left) + toFloat(right), nil
	case "-":
		return toFloat(left) - toFloat(right), nil
	case "*":
		return toFloat(left) * toFloat(right), nil
	case "/":
		r := toFloat(right)
		if r == 0 {
			return nil, fmt.Errorf("#DIV/0!")
		}
		return toFloat(left) / r, nil
	case "^":
		return math.Pow(toFloat(left), toFloat(right)), nil
	case "&":
		return toString(left) + toString(right), nil
	case "=":
		return compareValues(left, right) == 0, nil
	case "<>":
		return compareValues(left, right) != 0, nil
	case "<":
		return compareValues(left, right) < 0, nil
	case ">":
		return compareValues(left, right) > 0, nil
	case "<=":
		return compareValues(left, right) <= 0, nil
	case ">=":
		return compareValues(left, right) >= 0, nil
	default:
		return nil, fmt.Errorf("unknown operator: %s", op)
	}
}

func (e *Evaluator) evaluateUnaryOp(ctx context.Context, node *ASTNode) (interface{}, error) {
	if len(node.Children) != 1 {
		return nil, fmt.Errorf("unary operator requires 1 operand")
	}

	operand, err := e.Evaluate(ctx, node.Children[0])
	if err != nil {
		return nil, err
	}

	op := node.Value.(string)

	switch op {
	case "+":
		return toFloat(operand), nil
	case "-":
		return -toFloat(operand), nil
	case "%":
		return toFloat(operand) / 100.0, nil
	default:
		return nil, fmt.Errorf("unknown unary operator: %s", op)
	}
}

func (e *Evaluator) evaluateArray(ctx context.Context, node *ASTNode) (interface{}, error) {
	result := make([][]interface{}, len(node.Children))

	for i, row := range node.Children {
		if row.Type == NodeArray {
			result[i] = make([]interface{}, len(row.Children))
			for j, elem := range row.Children {
				val, err := e.Evaluate(ctx, elem)
				if err != nil {
					return nil, err
				}
				result[i][j] = val
			}
		} else {
			val, err := e.Evaluate(ctx, row)
			if err != nil {
				return nil, err
			}
			result[i] = []interface{}{val}
		}
	}

	return result, nil
}

// Helper functions

func toFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case bool:
		if val {
			return 1
		}
		return 0
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == math.Floor(val) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case bool:
		if val {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0
	case int:
		return val != 0
	case string:
		upper := strings.ToUpper(val)
		return upper == "TRUE" || upper == "YES" || upper == "1"
	default:
		return false
	}
}

func compareValues(a, b interface{}) int {
	// Handle nil
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Try numeric comparison
	aNum, aIsNum := toNumber(a)
	bNum, bIsNum := toNumber(b)
	if aIsNum && bIsNum {
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
		return 0
	}

	// String comparison
	aStr := toString(a)
	bStr := toString(b)
	return strings.Compare(strings.ToLower(aStr), strings.ToLower(bStr))
}

func toNumber(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// ParseCellRef parses a cell reference like "A1" or "$A$1" and returns row, col (0-indexed).
func ParseCellRef(ref string) (int, int, error) {
	ref = strings.ReplaceAll(ref, "$", "")
	ref = strings.ToUpper(ref)

	if len(ref) < 2 {
		return 0, 0, fmt.Errorf("invalid cell reference: %s", ref)
	}

	// Find where letters end
	i := 0
	for i < len(ref) && ref[i] >= 'A' && ref[i] <= 'Z' {
		i++
	}

	if i == 0 || i == len(ref) {
		return 0, 0, fmt.Errorf("invalid cell reference: %s", ref)
	}

	colStr := ref[:i]
	rowStr := ref[i:]

	// Parse column (A=0, B=1, ..., Z=25, AA=26, etc.)
	col := 0
	for _, c := range colStr {
		col = col*26 + int(c-'A') + 1
	}
	col-- // 0-indexed

	// Parse row
	row, err := strconv.Atoi(rowStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid row in cell reference: %s", ref)
	}
	row-- // 0-indexed

	return row, col, nil
}

// ParseRangeRef parses a range reference like "A1:B10" and returns start/end row/col (0-indexed).
func ParseRangeRef(rangeRef string) (int, int, int, int, error) {
	parts := strings.Split(rangeRef, ":")
	if len(parts) != 2 {
		return 0, 0, 0, 0, fmt.Errorf("invalid range reference: %s", rangeRef)
	}

	startRow, startCol, err := ParseCellRef(parts[0])
	if err != nil {
		return 0, 0, 0, 0, err
	}

	endRow, endCol, err := ParseCellRef(parts[1])
	if err != nil {
		return 0, 0, 0, 0, err
	}

	// Normalize so start <= end
	if startRow > endRow {
		startRow, endRow = endRow, startRow
	}
	if startCol > endCol {
		startCol, endCol = endCol, startCol
	}

	return startRow, startCol, endRow, endCol, nil
}

// ColToLetter converts a 0-indexed column number to letter(s).
func ColToLetter(col int) string {
	result := ""
	col++ // 1-indexed for calculation
	for col > 0 {
		col--
		result = string(rune('A'+col%26)) + result
		col /= 26
	}
	return result
}

// CellRefString returns a cell reference string like "A1" for the given row and col.
func CellRefString(row, col int) string {
	return ColToLetter(col) + strconv.Itoa(row+1)
}
