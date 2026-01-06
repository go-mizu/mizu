package formula

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// FunctionImpl is the signature for built-in functions.
type FunctionImpl func(args ...interface{}) (interface{}, error)

// Functions is the registry of built-in functions.
var Functions = map[string]FunctionImpl{
	// Math functions
	"SUM":      fnSum,
	"AVERAGE":  fnAverage,
	"MIN":      fnMin,
	"MAX":      fnMax,
	"COUNT":    fnCount,
	"COUNTA":   fnCountA,
	"COUNTBLANK": fnCountBlank,
	"ABS":      fnAbs,
	"ROUND":    fnRound,
	"ROUNDUP":  fnRoundUp,
	"ROUNDDOWN": fnRoundDown,
	"FLOOR":    fnFloor,
	"CEILING":  fnCeiling,
	"INT":      fnInt,
	"MOD":      fnMod,
	"POWER":    fnPower,
	"SQRT":     fnSqrt,
	"EXP":      fnExp,
	"LN":       fnLn,
	"LOG":      fnLog,
	"LOG10":    fnLog10,
	"PI":       fnPi,
	"RAND":     fnRand,
	"RANDBETWEEN": fnRandBetween,
	"SIGN":     fnSign,
	"PRODUCT":  fnProduct,
	"SUMPRODUCT": fnSumProduct,
	"SUMSQ":    fnSumSq,

	// Trigonometric functions
	"SIN":      fnSin,
	"COS":      fnCos,
	"TAN":      fnTan,
	"ASIN":     fnAsin,
	"ACOS":     fnAcos,
	"ATAN":     fnAtan,
	"ATAN2":    fnAtan2,
	"RADIANS":  fnRadians,
	"DEGREES":  fnDegrees,

	// Text functions
	"CONCATENATE": fnConcatenate,
	"CONCAT":   fnConcatenate,
	"LEFT":     fnLeft,
	"RIGHT":    fnRight,
	"MID":      fnMid,
	"LEN":      fnLen,
	"LOWER":    fnLower,
	"UPPER":    fnUpper,
	"PROPER":   fnProper,
	"TRIM":     fnTrim,
	"SUBSTITUTE": fnSubstitute,
	"REPLACE":  fnReplace,
	"REPT":     fnRept,
	"FIND":     fnFind,
	"SEARCH":   fnSearch,
	"TEXT":     fnText,
	"VALUE":    fnValue,
	"TEXTJOIN": fnTextJoin,
	"CHAR":     fnChar,
	"CODE":     fnCode,
	"CLEAN":    fnClean,
	"T":        fnT,
	"N":        fnN,

	// Logical functions
	"IF":       fnIf,
	"AND":      fnAnd,
	"OR":       fnOr,
	"NOT":      fnNot,
	"XOR":      fnXor,
	"IFS":      fnIfs,
	"SWITCH":   fnSwitch,
	"IFERROR":  fnIfError,
	"IFNA":     fnIfNA,
	"TRUE":     fnTrue,
	"FALSE":    fnFalse,

	// Lookup functions
	"VLOOKUP":  fnVlookup,
	"HLOOKUP":  fnHlookup,
	"INDEX":    fnIndex,
	"MATCH":    fnMatch,
	"CHOOSE":   fnChoose,
	"LOOKUP":   fnLookup,

	// Statistical functions
	"MEDIAN":   fnMedian,
	"MODE":     fnMode,
	"STDEV":    fnStdev,
	"STDEVP":   fnStdevP,
	"VAR":      fnVar,
	"VARP":     fnVarP,
	"LARGE":    fnLarge,
	"SMALL":    fnSmall,
	"RANK":     fnRank,
	"PERCENTILE": fnPercentile,
	"QUARTILE": fnQuartile,
	"CORREL":   fnCorrel,

	// Conditional aggregates
	"SUMIF":    fnSumIf,
	"COUNTIF":  fnCountIf,
	"AVERAGEIF": fnAverageIf,
	"SUMIFS":   fnSumIfs,
	"COUNTIFS": fnCountIfs,
	"AVERAGEIFS": fnAverageIfs,

	// Date/Time functions
	"NOW":      fnNow,
	"TODAY":    fnToday,
	"DATE":     fnDate,
	"YEAR":     fnYear,
	"MONTH":    fnMonth,
	"DAY":      fnDay,
	"HOUR":     fnHour,
	"MINUTE":   fnMinute,
	"SECOND":   fnSecond,
	"WEEKDAY":  fnWeekday,
	"DATEDIF":  fnDateDif,
	"EOMONTH":  fnEomonth,
	"EDATE":    fnEdate,
	"DAYS":     fnDays,
	"NETWORKDAYS": fnNetworkDays,

	// Information functions
	"ISBLANK":  fnIsBlank,
	"ISERROR":  fnIsError,
	"ISNA":     fnIsNA,
	"ISNUMBER": fnIsNumber,
	"ISTEXT":   fnIsText,
	"ISLOGICAL": fnIsLogical,
	"TYPE":     fnType,
	"NA":       fnNA,

	// Financial functions
	"PMT":      fnPmt,
	"PPMT":     fnPpmt,
	"IPMT":     fnIpmt,
	"PV":       fnPv,
	"FV":       fnFv,
	"NPV":      fnNpv,
	"IRR":      fnIrr,
	"RATE":     fnRate,
	"NPER":     fnNper,
}

// Math functions

func fnSum(args ...interface{}) (interface{}, error) {
	sum := 0.0
	for _, arg := range args {
		sum += sumValues(arg)
	}
	return sum, nil
}

func fnAverage(args ...interface{}) (interface{}, error) {
	sum := 0.0
	count := 0
	for _, arg := range args {
		s, c := sumCountValues(arg)
		sum += s
		count += c
	}
	if count == 0 {
		return nil, fmt.Errorf("#DIV/0!")
	}
	return sum / float64(count), nil
}

func fnMin(args ...interface{}) (interface{}, error) {
	result := math.MaxFloat64
	found := false
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if n, ok := toNumber(v); ok {
				if n < result {
					result = n
				}
				found = true
			}
		}
	}
	if !found {
		return 0.0, nil
	}
	return result, nil
}

func fnMax(args ...interface{}) (interface{}, error) {
	result := -math.MaxFloat64
	found := false
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if n, ok := toNumber(v); ok {
				if n > result {
					result = n
				}
				found = true
			}
		}
	}
	if !found {
		return 0.0, nil
	}
	return result, nil
}

func fnCount(args ...interface{}) (interface{}, error) {
	count := 0
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if _, ok := toNumber(v); ok {
				count++
			}
		}
	}
	return float64(count), nil
}

func fnCountA(args ...interface{}) (interface{}, error) {
	count := 0
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if v != nil && v != "" {
				count++
			}
		}
	}
	return float64(count), nil
}

func fnCountBlank(args ...interface{}) (interface{}, error) {
	count := 0
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if v == nil || v == "" {
				count++
			}
		}
	}
	return float64(count), nil
}

func fnAbs(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("ABS requires 1 argument")
	}
	return math.Abs(toFloat(args[0])), nil
}

func fnRound(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("ROUND requires at least 1 argument")
	}
	num := toFloat(args[0])
	digits := 0
	if len(args) > 1 {
		digits = int(toFloat(args[1]))
	}
	mult := math.Pow(10, float64(digits))
	return math.Round(num*mult) / mult, nil
}

func fnRoundUp(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("ROUNDUP requires at least 1 argument")
	}
	num := toFloat(args[0])
	digits := 0
	if len(args) > 1 {
		digits = int(toFloat(args[1]))
	}
	mult := math.Pow(10, float64(digits))
	if num >= 0 {
		return math.Ceil(num*mult) / mult, nil
	}
	return math.Floor(num*mult) / mult, nil
}

func fnRoundDown(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("ROUNDDOWN requires at least 1 argument")
	}
	num := toFloat(args[0])
	digits := 0
	if len(args) > 1 {
		digits = int(toFloat(args[1]))
	}
	mult := math.Pow(10, float64(digits))
	return math.Trunc(num*mult) / mult, nil
}

func fnFloor(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("FLOOR requires at least 1 argument")
	}
	num := toFloat(args[0])
	significance := 1.0
	if len(args) > 1 {
		significance = toFloat(args[1])
	}
	if significance == 0 {
		return 0.0, nil
	}
	return math.Floor(num/significance) * significance, nil
}

func fnCeiling(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("CEILING requires at least 1 argument")
	}
	num := toFloat(args[0])
	significance := 1.0
	if len(args) > 1 {
		significance = toFloat(args[1])
	}
	if significance == 0 {
		return 0.0, nil
	}
	return math.Ceil(num/significance) * significance, nil
}

func fnInt(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("INT requires 1 argument")
	}
	return math.Floor(toFloat(args[0])), nil
}

func fnMod(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("MOD requires 2 arguments")
	}
	divisor := toFloat(args[1])
	if divisor == 0 {
		return nil, fmt.Errorf("#DIV/0!")
	}
	return math.Mod(toFloat(args[0]), divisor), nil
}

func fnPower(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("POWER requires 2 arguments")
	}
	return math.Pow(toFloat(args[0]), toFloat(args[1])), nil
}

func fnSqrt(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("SQRT requires 1 argument")
	}
	num := toFloat(args[0])
	if num < 0 {
		return nil, fmt.Errorf("#NUM!")
	}
	return math.Sqrt(num), nil
}

func fnExp(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("EXP requires 1 argument")
	}
	return math.Exp(toFloat(args[0])), nil
}

func fnLn(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("LN requires 1 argument")
	}
	num := toFloat(args[0])
	if num <= 0 {
		return nil, fmt.Errorf("#NUM!")
	}
	return math.Log(num), nil
}

func fnLog(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("LOG requires at least 1 argument")
	}
	num := toFloat(args[0])
	if num <= 0 {
		return nil, fmt.Errorf("#NUM!")
	}
	base := 10.0
	if len(args) > 1 {
		base = toFloat(args[1])
	}
	if base <= 0 || base == 1 {
		return nil, fmt.Errorf("#NUM!")
	}
	return math.Log(num) / math.Log(base), nil
}

func fnLog10(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("LOG10 requires 1 argument")
	}
	num := toFloat(args[0])
	if num <= 0 {
		return nil, fmt.Errorf("#NUM!")
	}
	return math.Log10(num), nil
}

func fnPi(args ...interface{}) (interface{}, error) {
	return math.Pi, nil
}

func fnRand(args ...interface{}) (interface{}, error) {
	return float64(time.Now().UnixNano()%1000000) / 1000000.0, nil
}

func fnRandBetween(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("RANDBETWEEN requires 2 arguments")
	}
	bottom := int(toFloat(args[0]))
	top := int(toFloat(args[1]))
	if bottom > top {
		return nil, fmt.Errorf("#NUM!")
	}
	return float64(bottom + int(time.Now().UnixNano())%(top-bottom+1)), nil
}

func fnSign(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("SIGN requires 1 argument")
	}
	num := toFloat(args[0])
	if num > 0 {
		return 1.0, nil
	} else if num < 0 {
		return -1.0, nil
	}
	return 0.0, nil
}

func fnProduct(args ...interface{}) (interface{}, error) {
	product := 1.0
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if n, ok := toNumber(v); ok {
				product *= n
			}
		}
	}
	return product, nil
}

func fnSumProduct(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return 0.0, nil
	}

	// Get first array
	arrays := make([][][]interface{}, len(args))
	for i, arg := range args {
		if arr, ok := arg.([][]interface{}); ok {
			arrays[i] = arr
		} else {
			arrays[i] = [][]interface{}{{arg}}
		}
	}

	// Check dimensions match
	rows := len(arrays[0])
	cols := 0
	if rows > 0 {
		cols = len(arrays[0][0])
	}

	sum := 0.0
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			product := 1.0
			for _, arr := range arrays {
				if r < len(arr) && c < len(arr[r]) {
					product *= toFloat(arr[r][c])
				}
			}
			sum += product
		}
	}

	return sum, nil
}

func fnSumSq(args ...interface{}) (interface{}, error) {
	sum := 0.0
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if n, ok := toNumber(v); ok {
				sum += n * n
			}
		}
	}
	return sum, nil
}

// Trigonometric functions

func fnSin(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("SIN requires 1 argument")
	}
	return math.Sin(toFloat(args[0])), nil
}

func fnCos(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("COS requires 1 argument")
	}
	return math.Cos(toFloat(args[0])), nil
}

func fnTan(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("TAN requires 1 argument")
	}
	return math.Tan(toFloat(args[0])), nil
}

func fnAsin(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("ASIN requires 1 argument")
	}
	return math.Asin(toFloat(args[0])), nil
}

func fnAcos(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("ACOS requires 1 argument")
	}
	return math.Acos(toFloat(args[0])), nil
}

func fnAtan(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("ATAN requires 1 argument")
	}
	return math.Atan(toFloat(args[0])), nil
}

func fnAtan2(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("ATAN2 requires 2 arguments")
	}
	return math.Atan2(toFloat(args[0]), toFloat(args[1])), nil
}

func fnRadians(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("RADIANS requires 1 argument")
	}
	return toFloat(args[0]) * math.Pi / 180.0, nil
}

func fnDegrees(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("DEGREES requires 1 argument")
	}
	return toFloat(args[0]) * 180.0 / math.Pi, nil
}

// Text functions

func fnConcatenate(args ...interface{}) (interface{}, error) {
	var result strings.Builder
	for _, arg := range args {
		result.WriteString(toString(arg))
	}
	return result.String(), nil
}

func fnLeft(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("LEFT requires at least 1 argument")
	}
	text := toString(args[0])
	numChars := 1
	if len(args) > 1 {
		numChars = int(toFloat(args[1]))
	}
	if numChars < 0 {
		return nil, fmt.Errorf("#VALUE!")
	}
	if numChars > len(text) {
		return text, nil
	}
	return text[:numChars], nil
}

func fnRight(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("RIGHT requires at least 1 argument")
	}
	text := toString(args[0])
	numChars := 1
	if len(args) > 1 {
		numChars = int(toFloat(args[1]))
	}
	if numChars < 0 {
		return nil, fmt.Errorf("#VALUE!")
	}
	if numChars > len(text) {
		return text, nil
	}
	return text[len(text)-numChars:], nil
}

func fnMid(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("MID requires 3 arguments")
	}
	text := toString(args[0])
	startNum := int(toFloat(args[1]))
	numChars := int(toFloat(args[2]))

	if startNum < 1 || numChars < 0 {
		return nil, fmt.Errorf("#VALUE!")
	}

	start := startNum - 1
	if start >= len(text) {
		return "", nil
	}

	end := start + numChars
	if end > len(text) {
		end = len(text)
	}

	return text[start:end], nil
}

func fnLen(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("LEN requires 1 argument")
	}
	return float64(len(toString(args[0]))), nil
}

func fnLower(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("LOWER requires 1 argument")
	}
	return strings.ToLower(toString(args[0])), nil
}

func fnUpper(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("UPPER requires 1 argument")
	}
	return strings.ToUpper(toString(args[0])), nil
}

func fnProper(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("PROPER requires 1 argument")
	}
	return strings.Title(strings.ToLower(toString(args[0]))), nil
}

func fnTrim(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("TRIM requires 1 argument")
	}
	return strings.TrimSpace(toString(args[0])), nil
}

func fnSubstitute(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("SUBSTITUTE requires at least 3 arguments")
	}
	text := toString(args[0])
	oldText := toString(args[1])
	newText := toString(args[2])

	if len(args) > 3 {
		instance := int(toFloat(args[3]))
		count := 0
		result := text
		for i := 0; i < len(result); {
			idx := strings.Index(result[i:], oldText)
			if idx < 0 {
				break
			}
			count++
			if count == instance {
				result = result[:i+idx] + newText + result[i+idx+len(oldText):]
				break
			}
			i += idx + len(oldText)
		}
		return result, nil
	}

	return strings.ReplaceAll(text, oldText, newText), nil
}

func fnReplace(args ...interface{}) (interface{}, error) {
	if len(args) < 4 {
		return nil, fmt.Errorf("REPLACE requires 4 arguments")
	}
	text := toString(args[0])
	startNum := int(toFloat(args[1]))
	numChars := int(toFloat(args[2]))
	newText := toString(args[3])

	if startNum < 1 {
		return nil, fmt.Errorf("#VALUE!")
	}

	start := startNum - 1
	if start > len(text) {
		start = len(text)
	}

	end := start + numChars
	if end > len(text) {
		end = len(text)
	}

	return text[:start] + newText + text[end:], nil
}

func fnRept(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("REPT requires 2 arguments")
	}
	text := toString(args[0])
	times := int(toFloat(args[1]))
	if times < 0 {
		return nil, fmt.Errorf("#VALUE!")
	}
	return strings.Repeat(text, times), nil
}

func fnFind(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("FIND requires at least 2 arguments")
	}
	findText := toString(args[0])
	withinText := toString(args[1])
	startNum := 1
	if len(args) > 2 {
		startNum = int(toFloat(args[2]))
	}
	if startNum < 1 {
		return nil, fmt.Errorf("#VALUE!")
	}

	idx := strings.Index(withinText[startNum-1:], findText)
	if idx < 0 {
		return nil, fmt.Errorf("#VALUE!")
	}
	return float64(idx + startNum), nil
}

func fnSearch(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("SEARCH requires at least 2 arguments")
	}
	findText := strings.ToLower(toString(args[0]))
	withinText := strings.ToLower(toString(args[1]))
	startNum := 1
	if len(args) > 2 {
		startNum = int(toFloat(args[2]))
	}
	if startNum < 1 {
		return nil, fmt.Errorf("#VALUE!")
	}

	idx := strings.Index(withinText[startNum-1:], findText)
	if idx < 0 {
		return nil, fmt.Errorf("#VALUE!")
	}
	return float64(idx + startNum), nil
}

func fnText(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("TEXT requires 2 arguments")
	}
	value := toFloat(args[0])
	format := toString(args[1])

	// Basic format support
	if strings.Contains(format, "%") {
		return fmt.Sprintf("%.0f%%", value*100), nil
	}
	if strings.Contains(format, "$") {
		return fmt.Sprintf("$%.2f", value), nil
	}
	if strings.Contains(format, ".00") {
		return fmt.Sprintf("%.2f", value), nil
	}

	return fmt.Sprintf("%v", value), nil
}

func fnValue(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("VALUE requires 1 argument")
	}
	return toFloat(args[0]), nil
}

func fnTextJoin(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("TEXTJOIN requires at least 3 arguments")
	}
	delimiter := toString(args[0])
	ignoreEmpty := toBool(args[1])

	var parts []string
	for i := 2; i < len(args); i++ {
		for _, v := range flattenValues(args[i]) {
			s := toString(v)
			if !ignoreEmpty || s != "" {
				parts = append(parts, s)
			}
		}
	}

	return strings.Join(parts, delimiter), nil
}

func fnChar(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("CHAR requires 1 argument")
	}
	code := int(toFloat(args[0]))
	if code < 1 || code > 255 {
		return nil, fmt.Errorf("#VALUE!")
	}
	return string(rune(code)), nil
}

func fnCode(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("CODE requires 1 argument")
	}
	text := toString(args[0])
	if len(text) == 0 {
		return nil, fmt.Errorf("#VALUE!")
	}
	return float64(text[0]), nil
}

func fnClean(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("CLEAN requires 1 argument")
	}
	text := toString(args[0])
	var result strings.Builder
	for _, r := range text {
		if r >= 32 {
			result.WriteRune(r)
		}
	}
	return result.String(), nil
}

func fnT(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return "", nil
	}
	if s, ok := args[0].(string); ok {
		return s, nil
	}
	return "", nil
}

func fnN(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return 0.0, nil
	}
	switch v := args[0].(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return 0.0, nil
	}
}

// Logical functions

func fnIf(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("IF requires at least 2 arguments")
	}
	condition := toBool(args[0])
	if condition {
		return args[1], nil
	}
	if len(args) > 2 {
		return args[2], nil
	}
	return false, nil
}

func fnAnd(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("AND requires at least 1 argument")
	}
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if !toBool(v) {
				return false, nil
			}
		}
	}
	return true, nil
}

func fnOr(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("OR requires at least 1 argument")
	}
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if toBool(v) {
				return true, nil
			}
		}
	}
	return false, nil
}

func fnNot(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("NOT requires 1 argument")
	}
	return !toBool(args[0]), nil
}

func fnXor(args ...interface{}) (interface{}, error) {
	count := 0
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if toBool(v) {
				count++
			}
		}
	}
	return count%2 == 1, nil
}

func fnIfs(args ...interface{}) (interface{}, error) {
	if len(args) < 2 || len(args)%2 != 0 {
		return nil, fmt.Errorf("IFS requires pairs of condition and value")
	}
	for i := 0; i < len(args); i += 2 {
		if toBool(args[i]) {
			return args[i+1], nil
		}
	}
	return nil, fmt.Errorf("#N/A")
}

func fnSwitch(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("SWITCH requires at least 3 arguments")
	}
	expression := args[0]
	for i := 1; i < len(args)-1; i += 2 {
		if compareValues(expression, args[i]) == 0 {
			return args[i+1], nil
		}
	}
	if len(args)%2 == 0 {
		return args[len(args)-1], nil // Default value
	}
	return nil, fmt.Errorf("#N/A")
}

func fnIfError(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("IFERROR requires 2 arguments")
	}
	// In actual evaluation, errors would be caught
	return args[0], nil
}

func fnIfNA(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("IFNA requires 2 arguments")
	}
	return args[0], nil
}

func fnTrue(args ...interface{}) (interface{}, error) {
	return true, nil
}

func fnFalse(args ...interface{}) (interface{}, error) {
	return false, nil
}

// Lookup functions (simplified implementations)

func fnVlookup(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("VLOOKUP requires at least 3 arguments")
	}

	lookup := args[0]
	table, ok := args[1].([][]interface{})
	if !ok {
		return nil, fmt.Errorf("#VALUE!")
	}
	colIndex := int(toFloat(args[2]))
	exactMatch := false
	if len(args) > 3 {
		exactMatch = !toBool(args[3])
	}

	if colIndex < 1 || colIndex > len(table[0]) {
		return nil, fmt.Errorf("#REF!")
	}

	for _, row := range table {
		if len(row) < colIndex {
			continue
		}
		if exactMatch {
			if compareValues(row[0], lookup) == 0 {
				return row[colIndex-1], nil
			}
		} else {
			if compareValues(row[0], lookup) <= 0 {
				return row[colIndex-1], nil
			}
		}
	}

	return nil, fmt.Errorf("#N/A")
}

func fnHlookup(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("HLOOKUP requires at least 3 arguments")
	}

	lookup := args[0]
	table, ok := args[1].([][]interface{})
	if !ok {
		return nil, fmt.Errorf("#VALUE!")
	}
	rowIndex := int(toFloat(args[2]))
	exactMatch := false
	if len(args) > 3 {
		exactMatch = !toBool(args[3])
	}

	if rowIndex < 1 || rowIndex > len(table) {
		return nil, fmt.Errorf("#REF!")
	}

	for col := 0; col < len(table[0]); col++ {
		if exactMatch {
			if compareValues(table[0][col], lookup) == 0 {
				return table[rowIndex-1][col], nil
			}
		} else {
			if compareValues(table[0][col], lookup) <= 0 {
				return table[rowIndex-1][col], nil
			}
		}
	}

	return nil, fmt.Errorf("#N/A")
}

func fnIndex(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("INDEX requires at least 2 arguments")
	}

	array, ok := args[0].([][]interface{})
	if !ok {
		return nil, fmt.Errorf("#VALUE!")
	}

	rowNum := int(toFloat(args[1]))
	colNum := 1
	if len(args) > 2 {
		colNum = int(toFloat(args[2]))
	}

	if rowNum < 1 || rowNum > len(array) || colNum < 1 || colNum > len(array[0]) {
		return nil, fmt.Errorf("#REF!")
	}

	return array[rowNum-1][colNum-1], nil
}

func fnMatch(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("MATCH requires at least 2 arguments")
	}

	lookup := args[0]
	values := flattenValues(args[1])
	matchType := 1
	if len(args) > 2 {
		matchType = int(toFloat(args[2]))
	}

	for i, v := range values {
		if matchType == 0 {
			if compareValues(v, lookup) == 0 {
				return float64(i + 1), nil
			}
		} else if matchType == 1 {
			if compareValues(v, lookup) == 0 {
				return float64(i + 1), nil
			}
		} else {
			if compareValues(v, lookup) == 0 {
				return float64(i + 1), nil
			}
		}
	}

	return nil, fmt.Errorf("#N/A")
}

func fnChoose(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("CHOOSE requires at least 2 arguments")
	}
	index := int(toFloat(args[0]))
	if index < 1 || index >= len(args) {
		return nil, fmt.Errorf("#VALUE!")
	}
	return args[index], nil
}

func fnLookup(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("LOOKUP requires at least 2 arguments")
	}
	// Simplified: delegate to VLOOKUP behavior
	return fnVlookup(append(args, float64(1), false)...)
}

// Statistical functions

func fnMedian(args ...interface{}) (interface{}, error) {
	values := []float64{}
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if n, ok := toNumber(v); ok {
				values = append(values, n)
			}
		}
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("#NUM!")
	}
	sort.Float64s(values)
	mid := len(values) / 2
	if len(values)%2 == 0 {
		return (values[mid-1] + values[mid]) / 2, nil
	}
	return values[mid], nil
}

func fnMode(args ...interface{}) (interface{}, error) {
	counts := make(map[float64]int)
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if n, ok := toNumber(v); ok {
				counts[n]++
			}
		}
	}
	if len(counts) == 0 {
		return nil, fmt.Errorf("#N/A")
	}

	var mode float64
	maxCount := 0
	for v, c := range counts {
		if c > maxCount {
			maxCount = c
			mode = v
		}
	}
	if maxCount == 1 {
		return nil, fmt.Errorf("#N/A")
	}
	return mode, nil
}

func fnStdev(args ...interface{}) (interface{}, error) {
	values := []float64{}
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if n, ok := toNumber(v); ok {
				values = append(values, n)
			}
		}
	}
	if len(values) < 2 {
		return nil, fmt.Errorf("#DIV/0!")
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values) - 1)

	return math.Sqrt(variance), nil
}

func fnStdevP(args ...interface{}) (interface{}, error) {
	values := []float64{}
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if n, ok := toNumber(v); ok {
				values = append(values, n)
			}
		}
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("#DIV/0!")
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values))

	return math.Sqrt(variance), nil
}

func fnVar(args ...interface{}) (interface{}, error) {
	values := []float64{}
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if n, ok := toNumber(v); ok {
				values = append(values, n)
			}
		}
	}
	if len(values) < 2 {
		return nil, fmt.Errorf("#DIV/0!")
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}

	return variance / float64(len(values)-1), nil
}

func fnVarP(args ...interface{}) (interface{}, error) {
	values := []float64{}
	for _, arg := range args {
		for _, v := range flattenValues(arg) {
			if n, ok := toNumber(v); ok {
				values = append(values, n)
			}
		}
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("#DIV/0!")
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}

	return variance / float64(len(values)), nil
}

func fnLarge(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("LARGE requires 2 arguments")
	}
	values := []float64{}
	for _, v := range flattenValues(args[0]) {
		if n, ok := toNumber(v); ok {
			values = append(values, n)
		}
	}
	k := int(toFloat(args[1]))
	if k < 1 || k > len(values) {
		return nil, fmt.Errorf("#NUM!")
	}
	sort.Float64s(values)
	return values[len(values)-k], nil
}

func fnSmall(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("SMALL requires 2 arguments")
	}
	values := []float64{}
	for _, v := range flattenValues(args[0]) {
		if n, ok := toNumber(v); ok {
			values = append(values, n)
		}
	}
	k := int(toFloat(args[1]))
	if k < 1 || k > len(values) {
		return nil, fmt.Errorf("#NUM!")
	}
	sort.Float64s(values)
	return values[k-1], nil
}

func fnRank(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("RANK requires at least 2 arguments")
	}
	number := toFloat(args[0])
	values := []float64{}
	for _, v := range flattenValues(args[1]) {
		if n, ok := toNumber(v); ok {
			values = append(values, n)
		}
	}
	order := 0 // descending
	if len(args) > 2 {
		order = int(toFloat(args[2]))
	}

	sort.Float64s(values)
	if order == 0 {
		// Reverse for descending
		for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
			values[i], values[j] = values[j], values[i]
		}
	}

	for i, v := range values {
		if v == number {
			return float64(i + 1), nil
		}
	}
	return nil, fmt.Errorf("#N/A")
}

func fnPercentile(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("PERCENTILE requires 2 arguments")
	}
	values := []float64{}
	for _, v := range flattenValues(args[0]) {
		if n, ok := toNumber(v); ok {
			values = append(values, n)
		}
	}
	k := toFloat(args[1])
	if k < 0 || k > 1 || len(values) == 0 {
		return nil, fmt.Errorf("#NUM!")
	}
	sort.Float64s(values)
	index := k * float64(len(values)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	if lower == upper {
		return values[lower], nil
	}
	return values[lower] + (index-float64(lower))*(values[upper]-values[lower]), nil
}

func fnQuartile(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("QUARTILE requires 2 arguments")
	}
	quart := int(toFloat(args[1]))
	if quart < 0 || quart > 4 {
		return nil, fmt.Errorf("#NUM!")
	}
	return fnPercentile(args[0], float64(quart)/4.0)
}

func fnCorrel(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("CORREL requires 2 arguments")
	}
	x := flattenValues(args[0])
	y := flattenValues(args[1])
	if len(x) != len(y) || len(x) == 0 {
		return nil, fmt.Errorf("#N/A")
	}

	n := float64(len(x))
	sumX, sumY, sumXY, sumX2, sumY2 := 0.0, 0.0, 0.0, 0.0, 0.0
	for i := range x {
		xi := toFloat(x[i])
		yi := toFloat(y[i])
		sumX += xi
		sumY += yi
		sumXY += xi * yi
		sumX2 += xi * xi
		sumY2 += yi * yi
	}

	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))
	if denominator == 0 {
		return nil, fmt.Errorf("#DIV/0!")
	}
	return numerator / denominator, nil
}

// Conditional aggregates (simplified)

func fnSumIf(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("SUMIF requires at least 2 arguments")
	}
	// Simplified implementation
	return fnSum(args...)
}

func fnCountIf(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("COUNTIF requires at least 2 arguments")
	}
	return fnCount(args...)
}

func fnAverageIf(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("AVERAGEIF requires at least 2 arguments")
	}
	return fnAverage(args...)
}

func fnSumIfs(args ...interface{}) (interface{}, error) {
	return fnSum(args...)
}

func fnCountIfs(args ...interface{}) (interface{}, error) {
	return fnCount(args...)
}

func fnAverageIfs(args ...interface{}) (interface{}, error) {
	return fnAverage(args...)
}

// Date/Time functions

func fnNow(args ...interface{}) (interface{}, error) {
	return float64(time.Now().Unix()) / 86400.0 + 25569.0, nil
}

func fnToday(args ...interface{}) (interface{}, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return float64(today.Unix()) / 86400.0 + 25569.0, nil
}

func fnDate(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("DATE requires 3 arguments")
	}
	year := int(toFloat(args[0]))
	month := time.Month(int(toFloat(args[1])))
	day := int(toFloat(args[2]))
	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return float64(date.Unix())/86400.0 + 25569.0, nil
}

func fnYear(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("YEAR requires 1 argument")
	}
	serial := toFloat(args[0])
	t := time.Unix(int64((serial-25569)*86400), 0).UTC()
	return float64(t.Year()), nil
}

func fnMonth(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("MONTH requires 1 argument")
	}
	serial := toFloat(args[0])
	t := time.Unix(int64((serial-25569)*86400), 0).UTC()
	return float64(t.Month()), nil
}

func fnDay(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("DAY requires 1 argument")
	}
	serial := toFloat(args[0])
	t := time.Unix(int64((serial-25569)*86400), 0).UTC()
	return float64(t.Day()), nil
}

func fnHour(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("HOUR requires 1 argument")
	}
	serial := toFloat(args[0])
	t := time.Unix(int64((serial-25569)*86400), 0).UTC()
	return float64(t.Hour()), nil
}

func fnMinute(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("MINUTE requires 1 argument")
	}
	serial := toFloat(args[0])
	t := time.Unix(int64((serial-25569)*86400), 0).UTC()
	return float64(t.Minute()), nil
}

func fnSecond(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("SECOND requires 1 argument")
	}
	serial := toFloat(args[0])
	t := time.Unix(int64((serial-25569)*86400), 0).UTC()
	return float64(t.Second()), nil
}

func fnWeekday(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("WEEKDAY requires 1 argument")
	}
	serial := toFloat(args[0])
	t := time.Unix(int64((serial-25569)*86400), 0).UTC()
	return float64(t.Weekday()) + 1, nil
}

func fnDateDif(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("DATEDIF requires 3 arguments")
	}
	start := time.Unix(int64((toFloat(args[0])-25569)*86400), 0).UTC()
	end := time.Unix(int64((toFloat(args[1])-25569)*86400), 0).UTC()
	unit := strings.ToUpper(toString(args[2]))

	switch unit {
	case "D":
		return math.Floor(end.Sub(start).Hours() / 24), nil
	case "M":
		return float64((end.Year()-start.Year())*12 + int(end.Month()) - int(start.Month())), nil
	case "Y":
		return float64(end.Year() - start.Year()), nil
	default:
		return nil, fmt.Errorf("#NUM!")
	}
}

func fnEomonth(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("EOMONTH requires 2 arguments")
	}
	start := time.Unix(int64((toFloat(args[0])-25569)*86400), 0).UTC()
	months := int(toFloat(args[1]))
	result := start.AddDate(0, months+1, -start.Day())
	return float64(result.Unix())/86400.0 + 25569.0, nil
}

func fnEdate(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("EDATE requires 2 arguments")
	}
	start := time.Unix(int64((toFloat(args[0])-25569)*86400), 0).UTC()
	months := int(toFloat(args[1]))
	result := start.AddDate(0, months, 0)
	return float64(result.Unix())/86400.0 + 25569.0, nil
}

func fnDays(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("DAYS requires 2 arguments")
	}
	end := toFloat(args[0])
	start := toFloat(args[1])
	return math.Floor(end - start), nil
}

func fnNetworkDays(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("NETWORKDAYS requires 2 arguments")
	}
	start := time.Unix(int64((toFloat(args[0])-25569)*86400), 0).UTC()
	end := time.Unix(int64((toFloat(args[1])-25569)*86400), 0).UTC()

	count := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			count++
		}
	}
	return float64(count), nil
}

// Information functions

func fnIsBlank(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return true, nil
	}
	return args[0] == nil || args[0] == "", nil
}

func fnIsError(args ...interface{}) (interface{}, error) {
	// In actual implementation, this would check for error values
	return false, nil
}

func fnIsNA(args ...interface{}) (interface{}, error) {
	return false, nil
}

func fnIsNumber(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return false, nil
	}
	_, ok := toNumber(args[0])
	return ok, nil
}

func fnIsText(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return false, nil
	}
	_, ok := args[0].(string)
	return ok, nil
}

func fnIsLogical(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return false, nil
	}
	_, ok := args[0].(bool)
	return ok, nil
}

func fnType(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return 1.0, nil
	}
	switch args[0].(type) {
	case float64, int:
		return 1.0, nil
	case string:
		return 2.0, nil
	case bool:
		return 4.0, nil
	default:
		return 1.0, nil
	}
}

func fnNA(args ...interface{}) (interface{}, error) {
	return nil, fmt.Errorf("#N/A")
}

// Financial functions (simplified)

func fnPmt(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("PMT requires 3 arguments")
	}
	rate := toFloat(args[0])
	nper := toFloat(args[1])
	pv := toFloat(args[2])
	fv := 0.0
	if len(args) > 3 {
		fv = toFloat(args[3])
	}

	if rate == 0 {
		return -(pv + fv) / nper, nil
	}

	return -rate * (pv * math.Pow(1+rate, nper) + fv) / (math.Pow(1+rate, nper) - 1), nil
}

func fnPpmt(args ...interface{}) (interface{}, error) {
	return fnPmt(args...)
}

func fnIpmt(args ...interface{}) (interface{}, error) {
	return fnPmt(args...)
}

func fnPv(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("PV requires 3 arguments")
	}
	rate := toFloat(args[0])
	nper := toFloat(args[1])
	pmt := toFloat(args[2])

	if rate == 0 {
		return -pmt * nper, nil
	}

	return -pmt * (1 - math.Pow(1+rate, -nper)) / rate, nil
}

func fnFv(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("FV requires 3 arguments")
	}
	rate := toFloat(args[0])
	nper := toFloat(args[1])
	pmt := toFloat(args[2])
	pv := 0.0
	if len(args) > 3 {
		pv = toFloat(args[3])
	}

	if rate == 0 {
		return -pv - pmt*nper, nil
	}

	return -pv*math.Pow(1+rate, nper) - pmt*(math.Pow(1+rate, nper)-1)/rate, nil
}

func fnNpv(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("NPV requires at least 2 arguments")
	}
	rate := toFloat(args[0])
	npv := 0.0
	for i := 1; i < len(args); i++ {
		npv += toFloat(args[i]) / math.Pow(1+rate, float64(i))
	}
	return npv, nil
}

func fnIrr(args ...interface{}) (interface{}, error) {
	// Simplified IRR using Newton-Raphson
	if len(args) < 1 {
		return nil, fmt.Errorf("IRR requires at least 1 argument")
	}
	values := flattenValues(args[0])

	rate := 0.1
	for iter := 0; iter < 100; iter++ {
		npv := 0.0
		dnpv := 0.0
		for i, v := range values {
			cf := toFloat(v)
			npv += cf / math.Pow(1+rate, float64(i))
			dnpv -= float64(i) * cf / math.Pow(1+rate, float64(i+1))
		}
		if math.Abs(dnpv) < 1e-10 {
			break
		}
		rate -= npv / dnpv
	}
	return rate, nil
}

func fnRate(args ...interface{}) (interface{}, error) {
	// Simplified
	return 0.05, nil
}

func fnNper(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("NPER requires 3 arguments")
	}
	rate := toFloat(args[0])
	pmt := toFloat(args[1])
	pv := toFloat(args[2])

	if rate == 0 {
		return -pv / pmt, nil
	}

	return math.Log(-pmt/(pmt+pv*rate)) / math.Log(1+rate), nil
}

// Helper functions

func sumValues(arg interface{}) float64 {
	sum := 0.0
	for _, v := range flattenValues(arg) {
		if n, ok := toNumber(v); ok {
			sum += n
		}
	}
	return sum
}

func sumCountValues(arg interface{}) (float64, int) {
	sum := 0.0
	count := 0
	for _, v := range flattenValues(arg) {
		if n, ok := toNumber(v); ok {
			sum += n
			count++
		}
	}
	return sum, count
}

func flattenValues(arg interface{}) []interface{} {
	if arg == nil {
		return nil
	}

	switch v := arg.(type) {
	case [][]interface{}:
		result := []interface{}{}
		for _, row := range v {
			result = append(result, row...)
		}
		return result
	case []interface{}:
		return v
	default:
		return []interface{}{v}
	}
}
