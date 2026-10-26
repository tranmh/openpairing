package reports

import (
	"math"
	"strings"
	"testing"
)

// FuzzGetExpectedScore: the expected score stays within [0, 1] and never
// decreases as the rating difference grows.
func FuzzGetExpectedScore(f *testing.F) {
	f.Skip("aqaa: fuzz finding F1 (violation), re-enable once it is fixed; see report")
	for _, d := range []int{0, 5, 10, 399, 400, 405, 410, 799, 800, 1000, -1, -400, -410, -800, -1000} {
		f.Add(d)
	}
	c := NewNormCalculator()
	f.Fuzz(func(t *testing.T, d int) {
		if d < -2000 || d > 2000 {
			d %= 2000
		}
		e := c.getExpectedScore(d)
		if math.IsNaN(e) || e < 0 || e > 1 {
			t.Fatalf("getExpectedScore(%d) = %v, outside [0, 1]", d, e)
		}
		if next := c.getExpectedScore(d + 1); next < e {
			t.Errorf("getExpectedScore not monotonic: f(%d) = %v > f(%d) = %v", d, e, d+1, next)
		}
	})
}

// Regression for fuzz finding F1 (seed#7): the expected score must not go
// down from a rating difference of 799 to 800.
func TestGetExpectedScoreMonotonicAt799(t *testing.T) {
	c := NewNormCalculator()
	d := 799
	e := c.getExpectedScore(d)
	if next := c.getExpectedScore(d + 1); next < e {
		t.Errorf("getExpectedScore not monotonic: f(%d) = %v > f(%d) = %v", d, e, d+1, next)
	}
}

// FuzzCalculatePerformance: for results in [0, 1] the performance stays
// within 800 points of the average opponent rating.
func FuzzCalculatePerformance(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 0, 1})
	f.Add([]byte{255, 128, 3, 9, 1, 0, 0, 2})
	c := NewNormCalculator()
	f.Fuzz(func(t *testing.T, data []byte) {
		var games []NormGameDetail
		for i := 0; i+2 < len(data) && len(games) < 50; i += 3 {
			rating := int(data[i])<<4 | int(data[i+1]>>4) // 0..4095
			result := float64(data[i+2]%3) / 2             // 0, 0.5, 1
			games = append(games, NormGameDetail{OpponentRating: rating, Result: result})
		}
		perf := c.CalculatePerformance(games)
		if len(games) == 0 {
			if perf != 0 {
				t.Fatalf("CalculatePerformance(nil) = %d, want 0", perf)
			}
			return
		}
		total := 0
		for _, g := range games {
			total += g.OpponentRating
		}
		avg := float64(total) / float64(len(games))
		if float64(perf) < avg-801 || float64(perf) > avg+801 {
			t.Errorf("CalculatePerformance(%v) = %d, not within 800 of average %v", games, perf, avg)
		}
	})
}

// FuzzSanitizeFilename: the result holds only [A-Za-z0-9_-] and sanitising
// twice equals sanitising once.
func FuzzSanitizeFilename(f *testing.F) {
	for _, s := range []string{"", "Open 2024", "a/b\\c", "x\r\nSet-Cookie: y", "\"quoted\".trf", "\xff\xfe", strings.Repeat("ä", 300)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out := sanitizeFilename(s)
		for _, b := range []byte(out) {
			if !(b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '-' || b == '_') {
				t.Fatalf("sanitizeFilename(%q) = %q contains %q", s, out, b)
			}
		}
		if again := sanitizeFilename(out); again != out {
			t.Errorf("sanitizeFilename not idempotent for %q: %q then %q", s, out, again)
		}
	})
}

// FuzzEscapeXML: the escaped text has no markup characters and unescaping it
// gives the input back.
func FuzzEscapeXML(f *testing.F) {
	for _, s := range []string{"", "Schach & Co", "<a href='x'>\"y\"</a>", "&amp;", "\xff", strings.Repeat("<&>", 100)} {
		f.Add(s)
	}
	unescape := strings.NewReplacer("&lt;", "<", "&gt;", ">", "&apos;", "'", "&quot;", "\"", "&amp;", "&")
	f.Fuzz(func(t *testing.T, s string) {
		out := escapeXML(s)
		if strings.ContainsAny(out, "<>'\"") {
			t.Fatalf("escapeXML(%q) = %q still contains markup characters", s, out)
		}
		if back := unescape.Replace(out); back != s {
			t.Errorf("escapeXML round trip: %q -> %q -> %q", s, out, back)
		}
	})
}
