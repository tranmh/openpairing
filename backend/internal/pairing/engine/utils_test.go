package engine

import (
	"testing"
)

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"line1\nline2\nline3", 3},
		{"single", 1},
		{"", 1}, // Empty string returns slice with one empty string
		{"a\nb", 2},
	}

	for _, tt := range tests {
		got := splitLines(tt.input)
		if len(got) != tt.want {
			t.Errorf("splitLines(%q) = %d lines, want %d", tt.input, len(got), tt.want)
		}
	}
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  hello  ", "hello"},
		{"hello", "hello"},
		{"  ", ""},
		{"\t\nhello\t\n", "hello"},
	}

	for _, tt := range tests {
		got := trimSpace(tt.input)
		if got != tt.want {
			t.Errorf("trimSpace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTrimLeftSpace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  hello  ", "hello  "},
		{"hello", "hello"},
		{"  ", ""},
		{"\t\nhello", "hello"},
	}

	for _, tt := range tests {
		got := trimLeftSpace(tt.input)
		if got != tt.want {
			t.Errorf("trimLeftSpace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input  string
		length int
		want   string
	}{
		{"abc", 5, "abc  "},
		{"abc", 3, "abc"},
		{"abcdef", 3, "abc"},
		{"", 3, "   "},
	}

	for _, tt := range tests {
		got := padRight(tt.input, tt.length)
		if got != tt.want {
			t.Errorf("padRight(%q, %d) = %q, want %q", tt.input, tt.length, got, tt.want)
		}
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		input  string
		length int
		want   string
	}{
		{"abc", 5, "  abc"},
		{"abc", 3, "abc"},
		{"abcdef", 3, "def"},
		{"", 3, "   "},
	}

	for _, tt := range tests {
		got := padLeft(tt.input, tt.length)
		if got != tt.want {
			t.Errorf("padLeft(%q, %d) = %q, want %q", tt.input, tt.length, got, tt.want)
		}
	}
}

func TestMinInt(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{-1, 1, -1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		got := minInt(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("minInt(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestMaxInt(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{1, 2, 2},
		{2, 1, 2},
		{5, 5, 5},
		{-1, 1, 1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		got := maxInt(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("maxInt(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestAbsInt(t *testing.T) {
	tests := []struct {
		a    int
		want int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-100, 100},
	}

	for _, tt := range tests {
		got := absInt(tt.a)
		if got != tt.want {
			t.Errorf("absInt(%d) = %d, want %d", tt.a, got, tt.want)
		}
	}
}

func TestContainsInt(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}

	if !containsInt(slice, 3) {
		t.Error("containsInt should find 3 in slice")
	}

	if containsInt(slice, 10) {
		t.Error("containsInt should not find 10 in slice")
	}

	if containsInt(nil, 1) {
		t.Error("containsInt should return false for nil slice")
	}
}

func TestRemoveInt(t *testing.T) {
	tests := []struct {
		slice  []int
		val    int
		wantLen int
	}{
		{[]int{1, 2, 3, 4, 5}, 3, 4},
		{[]int{1, 2, 3, 4, 5}, 10, 5},
		{[]int{1}, 1, 0},
		{[]int{}, 1, 0},
	}

	for _, tt := range tests {
		got := removeInt(tt.slice, tt.val)
		if len(got) != tt.wantLen {
			t.Errorf("removeInt(%v, %d) length = %d, want %d", tt.slice, tt.val, len(got), tt.wantLen)
		}
	}
}

func TestReverseInts(t *testing.T) {
	tests := []struct {
		input []int
		want  []int
	}{
		{[]int{1, 2, 3, 4, 5}, []int{5, 4, 3, 2, 1}},
		{[]int{1, 2}, []int{2, 1}},
		{[]int{1}, []int{1}},
		{[]int{}, []int{}},
	}

	for _, tt := range tests {
		input := make([]int, len(tt.input))
		copy(input, tt.input)
		reverseInts(input)

		for i, v := range input {
			if v != tt.want[i] {
				t.Errorf("reverseInts(%v)[%d] = %d, want %d", tt.input, i, v, tt.want[i])
			}
		}
	}
}

func TestCopyInts(t *testing.T) {
	original := []int{1, 2, 3, 4, 5}
	copied := copyInts(original)

	// Verify copy is equal
	for i, v := range original {
		if copied[i] != v {
			t.Errorf("copyInts mismatch at index %d: got %d, want %d", i, copied[i], v)
		}
	}

	// Verify it's a true copy (modifying copy doesn't affect original)
	copied[0] = 100
	if original[0] == 100 {
		t.Error("copyInts should create independent copy")
	}
}

func TestUniqueInts(t *testing.T) {
	tests := []struct {
		input   []int
		wantLen int
	}{
		{[]int{1, 2, 3, 4, 5}, 5},
		{[]int{1, 1, 2, 2, 3, 3}, 3},
		{[]int{1, 1, 1, 1, 1}, 1},
		{[]int{}, 0},
	}

	for _, tt := range tests {
		got := uniqueInts(tt.input)
		if len(got) != tt.wantLen {
			t.Errorf("uniqueInts(%v) length = %d, want %d", tt.input, len(got), tt.wantLen)
		}
	}
}

func TestSumFloats(t *testing.T) {
	tests := []struct {
		input []float64
		want  float64
	}{
		{[]float64{1.0, 2.0, 3.0}, 6.0},
		{[]float64{0.5, 0.5}, 1.0},
		{[]float64{}, 0.0},
		{[]float64{-1.0, 1.0}, 0.0},
	}

	for _, tt := range tests {
		got := sumFloats(tt.input)
		if got != tt.want {
			t.Errorf("sumFloats(%v) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestAvgFloats(t *testing.T) {
	tests := []struct {
		input []float64
		want  float64
	}{
		{[]float64{2.0, 4.0, 6.0}, 4.0},
		{[]float64{1.0, 1.0, 1.0, 1.0}, 1.0},
		{[]float64{}, 0.0},
		{[]float64{10.0}, 10.0},
	}

	for _, tt := range tests {
		got := avgFloats(tt.input)
		if got != tt.want {
			t.Errorf("avgFloats(%v) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestParseColorFromChar(t *testing.T) {
	tests := []struct {
		input byte
		want  Color
	}{
		{'w', ColorWhite},
		{'W', ColorWhite},
		{'b', ColorBlack},
		{'B', ColorBlack},
		{'-', ColorNone},
		{'x', ColorNone},
	}

	for _, tt := range tests {
		got := ParseColorFromChar(tt.input)
		if got != tt.want {
			t.Errorf("ParseColorFromChar(%c) = %s, want %s", tt.input, got, tt.want)
		}
	}
}

func TestColorToChar(t *testing.T) {
	tests := []struct {
		input Color
		want  byte
	}{
		{ColorWhite, 'w'},
		{ColorBlack, 'b'},
		{ColorNone, '-'},
	}

	for _, tt := range tests {
		got := ColorToChar(tt.input)
		if got != tt.want {
			t.Errorf("ColorToChar(%s) = %c, want %c", tt.input, got, tt.want)
		}
	}
}

func TestOppositeColor(t *testing.T) {
	tests := []struct {
		input Color
		want  Color
	}{
		{ColorWhite, ColorBlack},
		{ColorBlack, ColorWhite},
		{ColorNone, ColorNone},
	}

	for _, tt := range tests {
		got := OppositeColor(tt.input)
		if got != tt.want {
			t.Errorf("OppositeColor(%s) = %s, want %s", tt.input, got, tt.want)
		}
	}
}
