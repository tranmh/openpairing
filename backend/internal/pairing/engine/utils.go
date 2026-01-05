package engine

import (
	"strings"
	"unicode"
)

// splitLines splits a string into lines
func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	return strings.TrimSpace(s)
}

// trimLeftSpace removes leading whitespace
func trimLeftSpace(s string) string {
	return strings.TrimLeftFunc(s, unicode.IsSpace)
}

// padRight pads a string to the right with spaces
func padRight(s string, length int) string {
	if len(s) >= length {
		return s[:length]
	}
	return s + strings.Repeat(" ", length-len(s))
}

// padLeft pads a string to the left with spaces
func padLeft(s string, length int) string {
	if len(s) >= length {
		return s[len(s)-length:]
	}
	return strings.Repeat(" ", length-len(s)) + s
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt returns the maximum of two integers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// absInt returns the absolute value of an integer
func absInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// containsInt checks if a slice contains an integer
func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

// removeInt removes the first occurrence of val from slice
func removeInt(slice []int, val int) []int {
	for i, v := range slice {
		if v == val {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// shuffleInts shuffles a slice of integers in place using Fisher-Yates
// Note: Uses a deterministic seed based on the first element for reproducibility
func shuffleInts(slice []int) {
	n := len(slice)
	if n <= 1 {
		return
	}

	// Simple deterministic shuffle for reproducibility in tests
	seed := 0
	if n > 0 {
		seed = slice[0]
	}

	for i := n - 1; i > 0; i-- {
		j := (seed + i) % (i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// reverseInts reverses a slice of integers in place
func reverseInts(slice []int) {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// copyInts creates a copy of a slice of integers
func copyInts(slice []int) []int {
	result := make([]int, len(slice))
	copy(result, slice)
	return result
}

// uniqueInts returns a slice with duplicate integers removed
func uniqueInts(slice []int) []int {
	seen := make(map[int]bool)
	result := make([]int, 0, len(slice))
	for _, v := range slice {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// sumFloats returns the sum of a slice of floats
func sumFloats(slice []float64) float64 {
	sum := 0.0
	for _, v := range slice {
		sum += v
	}
	return sum
}

// avgFloats returns the average of a slice of floats
func avgFloats(slice []float64) float64 {
	if len(slice) == 0 {
		return 0
	}
	return sumFloats(slice) / float64(len(slice))
}

// ParseColorFromChar converts a character to Color
func ParseColorFromChar(c byte) Color {
	switch c {
	case 'w', 'W':
		return ColorWhite
	case 'b', 'B':
		return ColorBlack
	default:
		return ColorNone
	}
}

// ColorToChar converts a Color to character
func ColorToChar(c Color) byte {
	switch c {
	case ColorWhite:
		return 'w'
	case ColorBlack:
		return 'b'
	default:
		return '-'
	}
}

// OppositeColor returns the opposite color
func OppositeColor(c Color) Color {
	switch c {
	case ColorWhite:
		return ColorBlack
	case ColorBlack:
		return ColorWhite
	default:
		return ColorNone
	}
}
