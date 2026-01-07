package formula

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

func init() {
	// Register advanced functions
	Functions["XLOOKUP"] = fnXlookup
	Functions["XMATCH"] = fnXmatch
	Functions["HYPERLINK"] = fnHyperlink
	Functions["HSTACK"] = fnHstack
	Functions["VSTACK"] = fnVstack
	Functions["TAKE"] = fnTake
	Functions["DROP"] = fnDrop
	Functions["EXPAND"] = fnExpand
	Functions["CHOOSECOLS"] = fnChooseCols
	Functions["CHOOSEROWS"] = fnChooseRows
	Functions["TOCOL"] = fnToCol
	Functions["TOROW"] = fnToRow
	Functions["WRAPCOLS"] = fnWrapCols
	Functions["WRAPROWS"] = fnWrapRows
	Functions["TEXTSPLIT"] = fnTextSplit
	Functions["ARRAYTOTEXT"] = fnArrayToText
	Functions["VALUETOTEXT"] = fnValueToText

	// Statistical functions
	Functions["GEOMEAN"] = fnGeoMean
	Functions["HARMEAN"] = fnHarMean
	Functions["TRIMMEAN"] = fnTrimMean
	Functions["AVEDEV"] = fnAveDev
	Functions["DEVSQ"] = fnDevSq
	Functions["SKEW"] = fnSkew
	Functions["KURT"] = fnKurt

	// Engineering functions (base conversion)
	Functions["DEC2BIN"] = fnDec2Bin
	Functions["DEC2HEX"] = fnDec2Hex
	Functions["DEC2OCT"] = fnDec2Oct
	Functions["BIN2DEC"] = fnBin2Dec
	Functions["BIN2HEX"] = fnBin2Hex
	Functions["BIN2OCT"] = fnBin2Oct
	Functions["HEX2DEC"] = fnHex2Dec
	Functions["HEX2BIN"] = fnHex2Bin
	Functions["HEX2OCT"] = fnHex2Oct
	Functions["OCT2DEC"] = fnOct2Dec
	Functions["OCT2BIN"] = fnOct2Bin
	Functions["OCT2HEX"] = fnOct2Hex
}

// XLOOKUP - Searches a range or an array, and returns an item corresponding to the first match.
func fnXlookup(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("XLOOKUP requires at least 3 arguments")
	}

	searchKey := args[0]
	lookupRange := toArray2D(args[1])
	returnRange := toArray2D(args[2])

	// Default values for optional arguments
	missingValue := interface{}(nil)
	matchMode := 0    // 0 = exact match, -1 = exact or smaller, 1 = exact or larger, 2 = wildcard
	searchMode := 1   // 1 = first to last, -1 = last to first, 2 = binary ascending, -2 = binary descending

	if len(args) > 3 && args[3] != nil {
		missingValue = args[3]
	}
	if len(args) > 4 {
		matchMode = int(toFloat(args[4]))
	}
	if len(args) > 5 {
		searchMode = int(toFloat(args[5]))
	}

	// Determine if lookup is horizontal or vertical
	isVertical := len(lookupRange) > 1 || (len(lookupRange) == 1 && len(lookupRange[0]) == 1)

	var matchIndex int = -1

	if isVertical {
		// Vertical lookup
		for i := 0; i < len(lookupRange); i++ {
			if len(lookupRange[i]) == 0 {
				continue
			}
			lookupVal := lookupRange[i][0]
			if matchesValue(searchKey, lookupVal, matchMode) {
				matchIndex = i
				if searchMode >= 0 {
					break // First match for forward search
				}
			}
		}
	} else {
		// Horizontal lookup
		if len(lookupRange) > 0 {
			for i := 0; i < len(lookupRange[0]); i++ {
				lookupVal := lookupRange[0][i]
				if matchesValue(searchKey, lookupVal, matchMode) {
					matchIndex = i
					if searchMode >= 0 {
						break
					}
				}
			}
		}
	}

	if matchIndex == -1 {
		if missingValue != nil {
			return missingValue, nil
		}
		return nil, fmt.Errorf("#N/A")
	}

	// Return value from return range
	if isVertical {
		if matchIndex < len(returnRange) && len(returnRange[matchIndex]) > 0 {
			// If return range has multiple columns, return the whole row
			if len(returnRange[matchIndex]) > 1 {
				return returnRange[matchIndex], nil
			}
			return returnRange[matchIndex][0], nil
		}
	} else {
		if len(returnRange) > 0 && matchIndex < len(returnRange[0]) {
			return returnRange[0][matchIndex], nil
		}
	}

	return nil, fmt.Errorf("#N/A")
}

// XMATCH - Returns the relative position of an item in an array or range of cells.
func fnXmatch(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("XMATCH requires at least 2 arguments")
	}

	searchKey := args[0]
	lookupRange := toArray2D(args[1])

	matchMode := 0  // 0 = exact, -1 = exact or smaller, 1 = exact or larger, 2 = wildcard
	searchMode := 1 // 1 = first to last, -1 = last to first

	if len(args) > 2 {
		matchMode = int(toFloat(args[2]))
	}
	if len(args) > 3 {
		searchMode = int(toFloat(args[3]))
	}

	// Flatten the lookup range
	var values []interface{}
	for _, row := range lookupRange {
		values = append(values, row...)
	}

	if searchMode < 0 {
		// Reverse search
		for i := len(values) - 1; i >= 0; i-- {
			if matchesValue(searchKey, values[i], matchMode) {
				return float64(i + 1), nil // 1-indexed
			}
		}
	} else {
		// Forward search
		for i, val := range values {
			if matchesValue(searchKey, val, matchMode) {
				return float64(i + 1), nil // 1-indexed
			}
		}
	}

	return nil, fmt.Errorf("#N/A")
}

// HYPERLINK - Creates a hyperlink
func fnHyperlink(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("HYPERLINK requires at least 1 argument")
	}

	url := toString(args[0])
	label := url
	if len(args) > 1 {
		label = toString(args[1])
	}

	// Return as a map with url and label for frontend rendering
	return map[string]string{
		"url":   url,
		"label": label,
	}, nil
}

// HSTACK - Appends arrays horizontally
func fnHstack(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("HSTACK requires at least 1 argument")
	}

	var maxRows int
	var arrays [][][]interface{}

	for _, arg := range args {
		arr := toArray2D(arg)
		arrays = append(arrays, arr)
		if len(arr) > maxRows {
			maxRows = len(arr)
		}
	}

	result := make([][]interface{}, maxRows)
	for i := range result {
		for _, arr := range arrays {
			if i < len(arr) {
				result[i] = append(result[i], arr[i]...)
			} else if len(arr) > 0 {
				// Pad with nil for shorter arrays
				result[i] = append(result[i], make([]interface{}, len(arr[0]))...)
			}
		}
	}

	return result, nil
}

// VSTACK - Appends arrays vertically
func fnVstack(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("VSTACK requires at least 1 argument")
	}

	var result [][]interface{}

	for _, arg := range args {
		arr := toArray2D(arg)
		result = append(result, arr...)
	}

	return result, nil
}

// TAKE - Returns the first or last rows/columns from an array
func fnTake(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("TAKE requires at least 2 arguments")
	}

	arr := toArray2D(args[0])
	rows := int(toFloat(args[1]))
	cols := 0
	if len(args) > 2 {
		cols = int(toFloat(args[2]))
	}

	if len(arr) == 0 {
		return arr, nil
	}

	// Handle rows
	var result [][]interface{}
	if rows >= 0 {
		if rows > len(arr) {
			rows = len(arr)
		}
		result = arr[:rows]
	} else {
		rows = -rows
		if rows > len(arr) {
			rows = len(arr)
		}
		result = arr[len(arr)-rows:]
	}

	// Handle columns if specified
	if cols != 0 && len(result) > 0 {
		for i := range result {
			if cols >= 0 {
				if cols > len(result[i]) {
					cols = len(result[i])
				}
				result[i] = result[i][:cols]
			} else {
				c := -cols
				if c > len(result[i]) {
					c = len(result[i])
				}
				result[i] = result[i][len(result[i])-c:]
			}
		}
	}

	return result, nil
}

// DROP - Removes the first or last rows/columns from an array
func fnDrop(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("DROP requires at least 2 arguments")
	}

	arr := toArray2D(args[0])
	rows := int(toFloat(args[1]))
	cols := 0
	if len(args) > 2 {
		cols = int(toFloat(args[2]))
	}

	if len(arr) == 0 {
		return arr, nil
	}

	// Handle rows
	var result [][]interface{}
	if rows >= 0 {
		if rows >= len(arr) {
			return [][]interface{}{}, nil
		}
		result = arr[rows:]
	} else {
		rows = -rows
		if rows >= len(arr) {
			return [][]interface{}{}, nil
		}
		result = arr[:len(arr)-rows]
	}

	// Handle columns if specified
	if cols != 0 && len(result) > 0 {
		for i := range result {
			if cols >= 0 {
				if cols >= len(result[i]) {
					result[i] = []interface{}{}
				} else {
					result[i] = result[i][cols:]
				}
			} else {
				c := -cols
				if c >= len(result[i]) {
					result[i] = []interface{}{}
				} else {
					result[i] = result[i][:len(result[i])-c]
				}
			}
		}
	}

	return result, nil
}

// EXPAND - Expands an array to specified dimensions
func fnExpand(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("EXPAND requires at least 3 arguments")
	}

	arr := toArray2D(args[0])
	targetRows := int(toFloat(args[1]))
	targetCols := int(toFloat(args[2]))
	padWith := interface{}(nil)
	if len(args) > 3 {
		padWith = args[3]
	}

	result := make([][]interface{}, targetRows)
	for i := 0; i < targetRows; i++ {
		result[i] = make([]interface{}, targetCols)
		for j := 0; j < targetCols; j++ {
			if i < len(arr) && j < len(arr[i]) {
				result[i][j] = arr[i][j]
			} else {
				result[i][j] = padWith
			}
		}
	}

	return result, nil
}

// CHOOSECOLS - Returns specified columns from an array
func fnChooseCols(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("CHOOSECOLS requires at least 2 arguments")
	}

	arr := toArray2D(args[0])
	if len(arr) == 0 {
		return arr, nil
	}

	result := make([][]interface{}, len(arr))
	for i := range result {
		result[i] = []interface{}{}
	}

	for _, colArg := range args[1:] {
		col := int(toFloat(colArg)) - 1 // 1-indexed
		if col < 0 {
			col = len(arr[0]) + col + 1 // Negative indexing from end
		}
		for i := range arr {
			if col < len(arr[i]) {
				result[i] = append(result[i], arr[i][col])
			} else {
				result[i] = append(result[i], nil)
			}
		}
	}

	return result, nil
}

// CHOOSEROWS - Returns specified rows from an array
func fnChooseRows(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("CHOOSEROWS requires at least 2 arguments")
	}

	arr := toArray2D(args[0])
	if len(arr) == 0 {
		return arr, nil
	}

	var result [][]interface{}

	for _, rowArg := range args[1:] {
		row := int(toFloat(rowArg)) - 1 // 1-indexed
		if row < 0 {
			row = len(arr) + row + 1 // Negative indexing from end
		}
		if row >= 0 && row < len(arr) {
			result = append(result, arr[row])
		}
	}

	return result, nil
}

// TOCOL - Converts an array to a single column
func fnToCol(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("TOCOL requires at least 1 argument")
	}

	arr := toArray2D(args[0])
	ignore := 0 // 0 = keep all, 1 = ignore blanks, 2 = ignore errors, 3 = ignore both
	scanByCol := false

	if len(args) > 1 {
		ignore = int(toFloat(args[1]))
	}
	if len(args) > 2 {
		scanByCol = toBool(args[2])
	}

	var result [][]interface{}

	if scanByCol {
		// Scan by column first
		if len(arr) > 0 {
			for col := 0; col < len(arr[0]); col++ {
				for row := 0; row < len(arr); row++ {
					if col < len(arr[row]) {
						val := arr[row][col]
						if shouldIncludeValue(val, ignore) {
							result = append(result, []interface{}{val})
						}
					}
				}
			}
		}
	} else {
		// Scan by row first
		for _, row := range arr {
			for _, val := range row {
				if shouldIncludeValue(val, ignore) {
					result = append(result, []interface{}{val})
				}
			}
		}
	}

	return result, nil
}

// TOROW - Converts an array to a single row
func fnToRow(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("TOROW requires at least 1 argument")
	}

	arr := toArray2D(args[0])
	ignore := 0
	scanByCol := false

	if len(args) > 1 {
		ignore = int(toFloat(args[1]))
	}
	if len(args) > 2 {
		scanByCol = toBool(args[2])
	}

	var result []interface{}

	if scanByCol {
		if len(arr) > 0 {
			for col := 0; col < len(arr[0]); col++ {
				for row := 0; row < len(arr); row++ {
					if col < len(arr[row]) {
						val := arr[row][col]
						if shouldIncludeValue(val, ignore) {
							result = append(result, val)
						}
					}
				}
			}
		}
	} else {
		for _, row := range arr {
			for _, val := range row {
				if shouldIncludeValue(val, ignore) {
					result = append(result, val)
				}
			}
		}
	}

	return [][]interface{}{result}, nil
}

// WRAPCOLS - Wraps values into columns
func fnWrapCols(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("WRAPCOLS requires at least 2 arguments")
	}

	arr := toArray2D(args[0])
	wrapCount := int(toFloat(args[1]))
	padWith := interface{}(nil)
	if len(args) > 2 {
		padWith = args[2]
	}

	if wrapCount <= 0 {
		return nil, fmt.Errorf("#VALUE!")
	}

	// Flatten the array
	var flat []interface{}
	for _, row := range arr {
		flat = append(flat, row...)
	}

	// Calculate dimensions
	numCols := (len(flat) + wrapCount - 1) / wrapCount
	result := make([][]interface{}, wrapCount)

	for i := 0; i < wrapCount; i++ {
		result[i] = make([]interface{}, numCols)
		for j := 0; j < numCols; j++ {
			idx := j*wrapCount + i
			if idx < len(flat) {
				result[i][j] = flat[idx]
			} else {
				result[i][j] = padWith
			}
		}
	}

	return result, nil
}

// WRAPROWS - Wraps values into rows
func fnWrapRows(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("WRAPROWS requires at least 2 arguments")
	}

	arr := toArray2D(args[0])
	wrapCount := int(toFloat(args[1]))
	padWith := interface{}(nil)
	if len(args) > 2 {
		padWith = args[2]
	}

	if wrapCount <= 0 {
		return nil, fmt.Errorf("#VALUE!")
	}

	// Flatten the array
	var flat []interface{}
	for _, row := range arr {
		flat = append(flat, row...)
	}

	// Calculate dimensions
	numRows := (len(flat) + wrapCount - 1) / wrapCount
	result := make([][]interface{}, numRows)

	for i := 0; i < numRows; i++ {
		result[i] = make([]interface{}, wrapCount)
		for j := 0; j < wrapCount; j++ {
			idx := i*wrapCount + j
			if idx < len(flat) {
				result[i][j] = flat[idx]
			} else {
				result[i][j] = padWith
			}
		}
	}

	return result, nil
}

// TEXTSPLIT - Splits text into array
func fnTextSplit(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("TEXTSPLIT requires at least 2 arguments")
	}

	text := toString(args[0])
	colDelim := toString(args[1])
	rowDelim := ""
	if len(args) > 2 && args[2] != nil {
		rowDelim = toString(args[2])
	}

	if rowDelim != "" {
		rows := strings.Split(text, rowDelim)
		result := make([][]interface{}, len(rows))
		for i, row := range rows {
			cols := strings.Split(row, colDelim)
			result[i] = make([]interface{}, len(cols))
			for j, col := range cols {
				result[i][j] = col
			}
		}
		return result, nil
	}

	// Single row
	cols := strings.Split(text, colDelim)
	result := make([]interface{}, len(cols))
	for i, col := range cols {
		result[i] = col
	}
	return [][]interface{}{result}, nil
}

// ARRAYTOTEXT - Converts array to text
func fnArrayToText(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("ARRAYTOTEXT requires at least 1 argument")
	}

	arr := toArray2D(args[0])
	format := 0 // 0 = concise, 1 = strict
	if len(args) > 1 {
		format = int(toFloat(args[1]))
	}

	var result strings.Builder

	if format == 1 {
		// Strict format with braces
		result.WriteString("{")
		for i, row := range arr {
			if i > 0 {
				result.WriteString(";")
			}
			for j, val := range row {
				if j > 0 {
					result.WriteString(",")
				}
				if s, ok := val.(string); ok {
					result.WriteString("\"")
					result.WriteString(s)
					result.WriteString("\"")
				} else {
					result.WriteString(toString(val))
				}
			}
		}
		result.WriteString("}")
	} else {
		// Concise format with commas
		first := true
		for _, row := range arr {
			for _, val := range row {
				if !first {
					result.WriteString(", ")
				}
				first = false
				result.WriteString(toString(val))
			}
		}
	}

	return result.String(), nil
}

// VALUETOTEXT - Converts a value to text
func fnValueToText(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("VALUETOTEXT requires at least 1 argument")
	}

	format := 0 // 0 = concise, 1 = strict
	if len(args) > 1 {
		format = int(toFloat(args[1]))
	}

	val := args[0]
	if format == 1 {
		if s, ok := val.(string); ok {
			return "\"" + s + "\"", nil
		}
	}

	return toString(val), nil
}

// Statistical functions

func fnGeoMean(args ...interface{}) (interface{}, error) {
	values := flattenToNumbers(args)
	if len(values) == 0 {
		return nil, fmt.Errorf("#NUM!")
	}

	product := 1.0
	for _, v := range values {
		if v <= 0 {
			return nil, fmt.Errorf("#NUM!")
		}
		product *= v
	}

	return math.Pow(product, 1.0/float64(len(values))), nil
}

func fnHarMean(args ...interface{}) (interface{}, error) {
	values := flattenToNumbers(args)
	if len(values) == 0 {
		return nil, fmt.Errorf("#NUM!")
	}

	sum := 0.0
	for _, v := range values {
		if v <= 0 {
			return nil, fmt.Errorf("#NUM!")
		}
		sum += 1.0 / v
	}

	return float64(len(values)) / sum, nil
}

func fnTrimMean(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("TRIMMEAN requires 2 arguments")
	}

	values := flattenToNumbers([]interface{}{args[0]})
	percent := toFloat(args[1])

	if percent < 0 || percent >= 1 {
		return nil, fmt.Errorf("#NUM!")
	}

	// Sort values
	sort.Float64s(values)

	// Calculate number of values to exclude from each end
	n := len(values)
	exclude := int(float64(n) * percent / 2)

	if exclude*2 >= n {
		return nil, fmt.Errorf("#NUM!")
	}

	// Calculate mean of remaining values
	sum := 0.0
	for i := exclude; i < n-exclude; i++ {
		sum += values[i]
	}

	return sum / float64(n-2*exclude), nil
}

func fnAveDev(args ...interface{}) (interface{}, error) {
	values := flattenToNumbers(args)
	if len(values) == 0 {
		return 0.0, nil
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Calculate average deviation
	devSum := 0.0
	for _, v := range values {
		devSum += math.Abs(v - mean)
	}

	return devSum / float64(len(values)), nil
}

func fnDevSq(args ...interface{}) (interface{}, error) {
	values := flattenToNumbers(args)
	if len(values) == 0 {
		return 0.0, nil
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Calculate sum of squared deviations
	devSqSum := 0.0
	for _, v := range values {
		devSqSum += (v - mean) * (v - mean)
	}

	return devSqSum, nil
}

func fnSkew(args ...interface{}) (interface{}, error) {
	values := flattenToNumbers(args)
	n := len(values)
	if n < 3 {
		return nil, fmt.Errorf("#DIV/0!")
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(n)

	// Calculate standard deviation
	varSum := 0.0
	for _, v := range values {
		varSum += (v - mean) * (v - mean)
	}
	stdDev := math.Sqrt(varSum / float64(n-1))

	if stdDev == 0 {
		return nil, fmt.Errorf("#DIV/0!")
	}

	// Calculate skewness
	skewSum := 0.0
	for _, v := range values {
		skewSum += math.Pow((v-mean)/stdDev, 3)
	}

	return float64(n) / float64((n-1)*(n-2)) * skewSum, nil
}

func fnKurt(args ...interface{}) (interface{}, error) {
	values := flattenToNumbers(args)
	n := len(values)
	if n < 4 {
		return nil, fmt.Errorf("#DIV/0!")
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(n)

	// Calculate standard deviation
	varSum := 0.0
	for _, v := range values {
		varSum += (v - mean) * (v - mean)
	}
	stdDev := math.Sqrt(varSum / float64(n-1))

	if stdDev == 0 {
		return nil, fmt.Errorf("#DIV/0!")
	}

	// Calculate kurtosis
	kurtSum := 0.0
	for _, v := range values {
		kurtSum += math.Pow((v-mean)/stdDev, 4)
	}

	nf := float64(n)
	return (nf*(nf+1))/((nf-1)*(nf-2)*(nf-3))*kurtSum - 3*(nf-1)*(nf-1)/((nf-2)*(nf-3)), nil
}

// Engineering functions (base conversion)

func fnDec2Bin(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("DEC2BIN requires at least 1 argument")
	}
	num := int64(toFloat(args[0]))
	if num < -512 || num > 511 {
		return nil, fmt.Errorf("#NUM!")
	}

	if num < 0 {
		// Two's complement for negative numbers
		return fmt.Sprintf("%010b", uint16(num))[6:], nil
	}

	result := fmt.Sprintf("%b", num)
	if len(args) > 1 {
		places := int(toFloat(args[1]))
		if places < len(result) {
			return nil, fmt.Errorf("#NUM!")
		}
		result = fmt.Sprintf("%0*s", places, result)
	}
	return result, nil
}

func fnDec2Hex(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("DEC2HEX requires at least 1 argument")
	}
	num := int64(toFloat(args[0]))

	if num < 0 {
		return fmt.Sprintf("%010X", uint64(num))[2:], nil
	}

	result := fmt.Sprintf("%X", num)
	if len(args) > 1 {
		places := int(toFloat(args[1]))
		if places < len(result) {
			return nil, fmt.Errorf("#NUM!")
		}
		result = fmt.Sprintf("%0*s", places, result)
	}
	return result, nil
}

func fnDec2Oct(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("DEC2OCT requires at least 1 argument")
	}
	num := int64(toFloat(args[0]))
	if num < -536870912 || num > 536870911 {
		return nil, fmt.Errorf("#NUM!")
	}

	if num < 0 {
		return fmt.Sprintf("%010o", uint32(num)), nil
	}

	result := fmt.Sprintf("%o", num)
	if len(args) > 1 {
		places := int(toFloat(args[1]))
		if places < len(result) {
			return nil, fmt.Errorf("#NUM!")
		}
		result = fmt.Sprintf("%0*s", places, result)
	}
	return result, nil
}

func fnBin2Dec(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("BIN2DEC requires 1 argument")
	}
	bin := toString(args[0])

	// Check for valid binary
	for _, c := range bin {
		if c != '0' && c != '1' {
			return nil, fmt.Errorf("#NUM!")
		}
	}

	if len(bin) > 10 {
		return nil, fmt.Errorf("#NUM!")
	}

	// Handle negative (two's complement)
	if len(bin) == 10 && bin[0] == '1' {
		var num int64
		fmt.Sscanf(bin, "%b", &num)
		return float64(num - 1024), nil
	}

	var num int64
	fmt.Sscanf(bin, "%b", &num)
	return float64(num), nil
}

func fnBin2Hex(args ...interface{}) (interface{}, error) {
	dec, err := fnBin2Dec(args[:1]...)
	if err != nil {
		return nil, err
	}
	newArgs := []interface{}{dec}
	if len(args) > 1 {
		newArgs = append(newArgs, args[1])
	}
	return fnDec2Hex(newArgs...)
}

func fnBin2Oct(args ...interface{}) (interface{}, error) {
	dec, err := fnBin2Dec(args[:1]...)
	if err != nil {
		return nil, err
	}
	newArgs := []interface{}{dec}
	if len(args) > 1 {
		newArgs = append(newArgs, args[1])
	}
	return fnDec2Oct(newArgs...)
}

func fnHex2Dec(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("HEX2DEC requires 1 argument")
	}
	hex := strings.ToUpper(toString(args[0]))

	if len(hex) > 10 {
		return nil, fmt.Errorf("#NUM!")
	}

	var num int64
	_, err := fmt.Sscanf(hex, "%X", &num)
	if err != nil {
		return nil, fmt.Errorf("#NUM!")
	}

	// Handle negative (10-char hex starting with 8-F)
	if len(hex) == 10 && hex[0] >= '8' {
		return float64(num - 0x10000000000), nil
	}

	return float64(num), nil
}

func fnHex2Bin(args ...interface{}) (interface{}, error) {
	dec, err := fnHex2Dec(args[:1]...)
	if err != nil {
		return nil, err
	}
	newArgs := []interface{}{dec}
	if len(args) > 1 {
		newArgs = append(newArgs, args[1])
	}
	return fnDec2Bin(newArgs...)
}

func fnHex2Oct(args ...interface{}) (interface{}, error) {
	dec, err := fnHex2Dec(args[:1]...)
	if err != nil {
		return nil, err
	}
	newArgs := []interface{}{dec}
	if len(args) > 1 {
		newArgs = append(newArgs, args[1])
	}
	return fnDec2Oct(newArgs...)
}

func fnOct2Dec(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("OCT2DEC requires 1 argument")
	}
	oct := toString(args[0])

	if len(oct) > 10 {
		return nil, fmt.Errorf("#NUM!")
	}

	var num int64
	_, err := fmt.Sscanf(oct, "%o", &num)
	if err != nil {
		return nil, fmt.Errorf("#NUM!")
	}

	// Handle negative (10-char octal starting with 4-7)
	if len(oct) == 10 && oct[0] >= '4' {
		return float64(num - 0x40000000), nil
	}

	return float64(num), nil
}

func fnOct2Bin(args ...interface{}) (interface{}, error) {
	dec, err := fnOct2Dec(args[:1]...)
	if err != nil {
		return nil, err
	}
	newArgs := []interface{}{dec}
	if len(args) > 1 {
		newArgs = append(newArgs, args[1])
	}
	return fnDec2Bin(newArgs...)
}

func fnOct2Hex(args ...interface{}) (interface{}, error) {
	dec, err := fnOct2Dec(args[:1]...)
	if err != nil {
		return nil, err
	}
	newArgs := []interface{}{dec}
	if len(args) > 1 {
		newArgs = append(newArgs, args[1])
	}
	return fnDec2Hex(newArgs...)
}

// Helper functions

func toArray2D(v interface{}) [][]interface{} {
	if v == nil {
		return [][]interface{}{}
	}

	switch val := v.(type) {
	case [][]interface{}:
		return val
	case []interface{}:
		return [][]interface{}{val}
	default:
		return [][]interface{}{{v}}
	}
}

func matchesValue(search, lookup interface{}, matchMode int) bool {
	switch matchMode {
	case 0: // Exact match
		return compareValues(search, lookup) == 0
	case 2: // Wildcard match
		searchStr := toString(search)
		lookupStr := toString(lookup)
		// Convert wildcards to regex
		pattern := "^" + strings.ReplaceAll(strings.ReplaceAll(searchStr, "*", ".*"), "?", ".") + "$"
		matched, _ := matchPattern(pattern, lookupStr)
		return matched
	case -1: // Exact or next smaller
		cmp := compareValues(search, lookup)
		return cmp == 0 || cmp > 0
	case 1: // Exact or next larger
		cmp := compareValues(search, lookup)
		return cmp == 0 || cmp < 0
	default:
		return compareValues(search, lookup) == 0
	}
}

func matchPattern(pattern, s string) (bool, error) {
	// Simple pattern matching
	pattern = strings.ToLower(pattern)
	s = strings.ToLower(s)

	pi, si := 0, 0
	starIdx, matchIdx := -1, 0

	for si < len(s) {
		if pi < len(pattern) && (pattern[pi] == '?' || pattern[pi] == s[si]) {
			pi++
			si++
		} else if pi < len(pattern) && pattern[pi] == '*' {
			starIdx = pi
			matchIdx = si
			pi++
		} else if starIdx != -1 {
			pi = starIdx + 1
			matchIdx++
			si = matchIdx
		} else {
			return false, nil
		}
	}

	for pi < len(pattern) && pattern[pi] == '*' {
		pi++
	}

	return pi == len(pattern), nil
}

func shouldIncludeValue(val interface{}, ignore int) bool {
	isBlank := val == nil || val == ""
	isError := false
	if s, ok := val.(string); ok {
		isError = strings.HasPrefix(s, "#")
	}

	switch ignore {
	case 1: // Ignore blanks
		return !isBlank
	case 2: // Ignore errors
		return !isError
	case 3: // Ignore both
		return !isBlank && !isError
	default:
		return true
	}
}

func flattenToNumbers(args []interface{}) []float64 {
	var result []float64
	for _, arg := range args {
		for _, val := range flattenValues(arg) {
			if n, ok := toNumber(val); ok {
				result = append(result, n)
			}
		}
	}
	return result
}
