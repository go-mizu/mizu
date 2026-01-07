package formula

import (
	"math"
	"testing"
)

// almostEqual checks if two floats are within a tolerance
func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestXLOOKUP(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "exact match found",
			args: []interface{}{
				"B",                                           // search key
				[][]interface{}{{"A"}, {"B"}, {"C"}},          // lookup range (vertical)
				[][]interface{}{{100}, {200}, {300}},          // return range
			},
			expected: 200,
			wantErr:  false,
		},
		{
			name: "exact match not found with default",
			args: []interface{}{
				"D",
				[][]interface{}{{"A"}, {"B"}, {"C"}},
				[][]interface{}{{100}, {200}, {300}},
				"Not Found", // missing value
			},
			expected: "Not Found",
			wantErr:  false,
		},
		{
			name: "exact match not found without default",
			args: []interface{}{
				"D",
				[][]interface{}{{"A"}, {"B"}, {"C"}},
				[][]interface{}{{100}, {200}, {300}},
			},
			wantErr: true,
		},
		{
			name: "numeric match",
			args: []interface{}{
				2.0,
				[][]interface{}{{1.0}, {2.0}, {3.0}},
				[][]interface{}{{"One"}, {"Two"}, {"Three"}},
			},
			expected: "Two",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fnXlookup(tt.args...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestXMATCH(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected float64
		wantErr  bool
	}{
		{
			name: "exact match",
			args: []interface{}{
				"B",
				[][]interface{}{{"A"}, {"B"}, {"C"}},
			},
			expected: 2.0, // 1-indexed
			wantErr:  false,
		},
		{
			name: "not found",
			args: []interface{}{
				"D",
				[][]interface{}{{"A"}, {"B"}, {"C"}},
			},
			wantErr: true,
		},
		{
			name: "numeric match",
			args: []interface{}{
				50.0,
				[][]interface{}{{10.0}, {50.0}, {100.0}},
			},
			expected: 2.0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fnXmatch(tt.args...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHSTACK(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected [][]interface{}
	}{
		{
			name: "stack two single columns",
			args: []interface{}{
				[][]interface{}{{1}, {2}, {3}},
				[][]interface{}{{4}, {5}, {6}},
			},
			expected: [][]interface{}{{1, 4}, {2, 5}, {3, 6}},
		},
		{
			name: "stack multi-column arrays",
			args: []interface{}{
				[][]interface{}{{1, 2}, {3, 4}},
				[][]interface{}{{5, 6}, {7, 8}},
			},
			expected: [][]interface{}{{1, 2, 5, 6}, {3, 4, 7, 8}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fnHstack(tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			resultArr, ok := result.([][]interface{})
			if !ok {
				t.Fatalf("result is not [][]interface{}")
			}
			if len(resultArr) != len(tt.expected) {
				t.Fatalf("row count mismatch: got %d, want %d", len(resultArr), len(tt.expected))
			}
			for i, row := range resultArr {
				if len(row) != len(tt.expected[i]) {
					t.Errorf("row %d col count mismatch: got %d, want %d", i, len(row), len(tt.expected[i]))
					continue
				}
				for j, val := range row {
					if val != tt.expected[i][j] {
						t.Errorf("cell [%d][%d]: got %v, want %v", i, j, val, tt.expected[i][j])
					}
				}
			}
		})
	}
}

func TestVSTACK(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected [][]interface{}
	}{
		{
			name: "stack two rows",
			args: []interface{}{
				[][]interface{}{{1, 2, 3}},
				[][]interface{}{{4, 5, 6}},
			},
			expected: [][]interface{}{{1, 2, 3}, {4, 5, 6}},
		},
		{
			name: "stack multi-row arrays",
			args: []interface{}{
				[][]interface{}{{1, 2}, {3, 4}},
				[][]interface{}{{5, 6}},
			},
			expected: [][]interface{}{{1, 2}, {3, 4}, {5, 6}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fnVstack(tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			resultArr, ok := result.([][]interface{})
			if !ok {
				t.Fatalf("result is not [][]interface{}")
			}
			if len(resultArr) != len(tt.expected) {
				t.Fatalf("row count mismatch: got %d, want %d", len(resultArr), len(tt.expected))
			}
			for i, row := range resultArr {
				for j, val := range row {
					if val != tt.expected[i][j] {
						t.Errorf("cell [%d][%d]: got %v, want %v", i, j, val, tt.expected[i][j])
					}
				}
			}
		})
	}
}

func TestTAKE(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected [][]interface{}
	}{
		{
			name: "take first 2 rows",
			args: []interface{}{
				[][]interface{}{{1, 2}, {3, 4}, {5, 6}},
				2.0,
			},
			expected: [][]interface{}{{1, 2}, {3, 4}},
		},
		{
			name: "take last 2 rows",
			args: []interface{}{
				[][]interface{}{{1, 2}, {3, 4}, {5, 6}},
				-2.0,
			},
			expected: [][]interface{}{{3, 4}, {5, 6}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fnTake(tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			resultArr, ok := result.([][]interface{})
			if !ok {
				t.Fatalf("result is not [][]interface{}")
			}
			if len(resultArr) != len(tt.expected) {
				t.Fatalf("row count mismatch: got %d, want %d", len(resultArr), len(tt.expected))
			}
		})
	}
}

func TestDROP(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected [][]interface{}
	}{
		{
			name: "drop first 2 rows",
			args: []interface{}{
				[][]interface{}{{1, 2}, {3, 4}, {5, 6}},
				2.0,
			},
			expected: [][]interface{}{{5, 6}},
		},
		{
			name: "drop last 2 rows",
			args: []interface{}{
				[][]interface{}{{1, 2}, {3, 4}, {5, 6}},
				-2.0,
			},
			expected: [][]interface{}{{1, 2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fnDrop(tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			resultArr, ok := result.([][]interface{})
			if !ok {
				t.Fatalf("result is not [][]interface{}")
			}
			if len(resultArr) != len(tt.expected) {
				t.Fatalf("row count mismatch: got %d, want %d", len(resultArr), len(tt.expected))
			}
		})
	}
}

func TestTEXTSPLIT(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected [][]interface{}
	}{
		{
			name: "split by comma",
			args: []interface{}{
				"a,b,c",
				",",
			},
			expected: [][]interface{}{{"a", "b", "c"}},
		},
		{
			name: "split by comma and newline",
			args: []interface{}{
				"a,b\nc,d",
				",",
				"\n",
			},
			expected: [][]interface{}{{"a", "b"}, {"c", "d"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fnTextSplit(tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			resultArr, ok := result.([][]interface{})
			if !ok {
				t.Fatalf("result is not [][]interface{}")
			}
			if len(resultArr) != len(tt.expected) {
				t.Fatalf("row count mismatch: got %d, want %d", len(resultArr), len(tt.expected))
			}
			for i, row := range resultArr {
				for j, val := range row {
					if val != tt.expected[i][j] {
						t.Errorf("cell [%d][%d]: got %v, want %v", i, j, val, tt.expected[i][j])
					}
				}
			}
		})
	}
}

func TestGEOMEAN(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected float64
		wantErr  bool
	}{
		{
			name:     "simple geometric mean",
			args:     []interface{}{1.0, 4.0},
			expected: 2.0, // sqrt(1*4) = 2
			wantErr:  false,
		},
		{
			name:     "three values",
			args:     []interface{}{2.0, 4.0, 8.0},
			expected: 4.0, // cuberoot(2*4*8) = cuberoot(64) = 4
			wantErr:  false,
		},
		{
			name:    "zero value",
			args:    []interface{}{0.0, 4.0},
			wantErr: true, // Cannot have zero in geometric mean
		},
		{
			name:    "negative value",
			args:    []interface{}{-1.0, 4.0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fnGeoMean(tt.args...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			resultFloat, ok := result.(float64)
			if !ok {
				t.Errorf("result is not float64: %v", result)
				return
			}
			if !almostEqual(resultFloat, tt.expected, 1e-9) {
				t.Errorf("got %v, want %v", resultFloat, tt.expected)
			}
		})
	}
}

func TestDec2Bin(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "positive number",
			args:     []interface{}{10.0},
			expected: "1010",
			wantErr:  false,
		},
		{
			name:     "zero",
			args:     []interface{}{0.0},
			expected: "0",
			wantErr:  false,
		},
		{
			name:     "with places",
			args:     []interface{}{10.0, 8.0},
			expected: "00001010",
			wantErr:  false,
		},
		{
			name:    "out of range",
			args:    []interface{}{1000.0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fnDec2Bin(tt.args...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDec2Hex(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "positive number",
			args:     []interface{}{255.0},
			expected: "FF",
			wantErr:  false,
		},
		{
			name:     "with places",
			args:     []interface{}{255.0, 4.0},
			expected: "00FF",
			wantErr:  false,
		},
		{
			name:     "zero",
			args:     []interface{}{0.0},
			expected: "0",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fnDec2Hex(tt.args...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBin2Dec(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected float64
		wantErr  bool
	}{
		{
			name:     "positive number",
			args:     []interface{}{"1010"},
			expected: 10.0,
			wantErr:  false,
		},
		{
			name:     "zero",
			args:     []interface{}{"0"},
			expected: 0.0,
			wantErr:  false,
		},
		{
			name:    "invalid binary",
			args:    []interface{}{"123"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fnBin2Dec(tt.args...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHex2Dec(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected float64
		wantErr  bool
	}{
		{
			name:     "FF",
			args:     []interface{}{"FF"},
			expected: 255.0,
			wantErr:  false,
		},
		{
			name:     "lowercase",
			args:     []interface{}{"ff"},
			expected: 255.0,
			wantErr:  false,
		},
		{
			name:     "zero",
			args:     []interface{}{"0"},
			expected: 0.0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fnHex2Dec(tt.args...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
