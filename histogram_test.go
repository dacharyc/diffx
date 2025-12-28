package diffx

import (
	"reflect"
	"strings"
	"testing"
)

func TestIsStopword(t *testing.T) {
	tests := []struct {
		word string
		want bool
	}{
		// Stopwords
		{"the", true},
		{"a", true},
		{"an", true},
		{"in", true},
		{"on", true},
		{"to", true},
		{"for", true},
		{"of", true},
		{"with", true},
		{"and", true},
		{"or", true},
		{"is", true},
		{"are", true},
		{"be", true},

		// Non-stopwords
		{"fox", false},
		{"quick", false},
		{"function", false},
		{"main", false},
		{"", false},
		{"The", false}, // Case-sensitive
		{"THE", false},
	}

	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			got := isStopword(StringElement(tt.word))
			if got != tt.want {
				t.Errorf("isStopword(%q) = %v, want %v", tt.word, got, tt.want)
			}
		})
	}
}

func TestIsStopword_NonStringElement(t *testing.T) {
	// Verify behavior with non-stopword StringElement
	if isStopword(StringElement("notastopword")) {
		t.Error("expected non-stopword to return false")
	}

	// Empty string should not be a stopword
	if isStopword(StringElement("")) {
		t.Error("expected empty string to not be a stopword")
	}
}

func TestHistogramDiff_Empty(t *testing.T) {
	tests := []struct {
		name string
		a, b []Element
		want []DiffOp
	}{
		{
			name: "both empty",
			a:    []Element{},
			b:    []Element{},
			want: nil,
		},
		{
			name: "a empty",
			a:    []Element{},
			b:    toElements([]string{"x", "y"}),
			want: []DiffOp{{Type: Insert, AStart: 0, AEnd: 0, BStart: 0, BEnd: 2}},
		},
		{
			name: "b empty",
			a:    toElements([]string{"x", "y"}),
			b:    []Element{},
			want: []DiffOp{{Type: Delete, AStart: 0, AEnd: 2, BStart: 0, BEnd: 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := histogramDiff(tt.a, tt.b, nil)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("histogramDiff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHistogramDiff_Equal(t *testing.T) {
	a := toElements([]string{"a", "b", "c"})
	b := toElements([]string{"a", "b", "c"})

	got := histogramDiff(a, b, nil)

	if len(got) != 1 || got[0].Type != Equal {
		t.Errorf("expected single Equal op, got %v", got)
	}

	if got[0].AStart != 0 || got[0].AEnd != 3 || got[0].BStart != 0 || got[0].BEnd != 3 {
		t.Errorf("expected Equal 0-3, got %v", got[0])
	}
}

func TestHistogramDiff_CommonPrefixSuffix(t *testing.T) {
	a := toElements([]string{"prefix", "old", "suffix"})
	b := toElements([]string{"prefix", "new", "suffix"})

	got := histogramDiff(a, b, nil)

	// Should detect common prefix and suffix
	result := applyHistogramDiff([]string{"prefix", "old", "suffix"}, []string{"prefix", "new", "suffix"}, got)
	want := []string{"prefix", "new", "suffix"}

	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestHistogramDiff_StopwordFiltering(t *testing.T) {
	// Stopwords should not be chosen as anchors
	a := toElements([]string{"the", "quick", "fox"})
	b := toElements([]string{"the", "slow", "fox"})

	opts := defaultHistogramOptions()
	got := histogramDiff(a, b, opts)

	// "fox" should be the anchor, not "the"
	result := applyHistogramDiff([]string{"the", "quick", "fox"}, []string{"the", "slow", "fox"}, got)
	want := []string{"the", "slow", "fox"}

	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestHistogramDiff_LowFrequencyAnchor(t *testing.T) {
	// Should prefer low-frequency elements as anchors
	a := toElements([]string{"common", "common", "unique", "common", "common"})
	b := toElements([]string{"other", "other", "unique", "other", "other"})

	got := histogramDiff(a, b, nil)

	// Verify "unique" is preserved
	result := applyHistogramDiff(
		[]string{"common", "common", "unique", "common", "common"},
		[]string{"other", "other", "unique", "other", "other"},
		got,
	)
	want := []string{"other", "other", "unique", "other", "other"}

	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}

	// Check that "unique" appears in an Equal operation
	uniquePreserved := false
	for _, op := range got {
		if op.Type == Equal && op.AStart <= 2 && op.AEnd > 2 {
			uniquePreserved = true
			break
		}
	}
	if !uniquePreserved {
		t.Error("expected 'unique' to be preserved as anchor")
	}
}

func TestDiffHistogram(t *testing.T) {
	a := []string{"the", "quick", "brown", "fox"}
	b := []string{"a", "slow", "red", "fox"}

	ops := DiffHistogram(a, b)

	result := applyDiff(a, b, ops)
	if !reflect.DeepEqual(result, b) {
		t.Errorf("DiffHistogram: got %v, want %v", result, b)
	}
}

func TestDiffElementsHistogram(t *testing.T) {
	a := toElements([]string{"one", "two", "three"})
	b := toElements([]string{"one", "TWO", "three"})

	ops := DiffElementsHistogram(a, b)

	// Apply and verify
	var result []string
	for _, op := range ops {
		switch op.Type {
		case Equal:
			for i := op.AStart; i < op.AEnd; i++ {
				result = append(result, string(a[i].(StringElement)))
			}
		case Insert:
			for i := op.BStart; i < op.BEnd; i++ {
				result = append(result, string(b[i].(StringElement)))
			}
		}
	}

	want := []string{"one", "TWO", "three"}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("DiffElementsHistogram: got %v, want %v", result, want)
	}
}

func TestHistogramDiff_MyersFallback(t *testing.T) {
	// When all elements are stopwords or high-frequency, should fall back to Myers
	a := toElements([]string{"the", "a", "an", "in"})
	b := toElements([]string{"the", "to", "for", "in"})

	opts := defaultHistogramOptions()
	got := histogramDiff(a, b, opts)

	result := applyHistogramDiff(
		[]string{"the", "a", "an", "in"},
		[]string{"the", "to", "for", "in"},
		got,
	)
	want := []string{"the", "to", "for", "in"}

	if !reflect.DeepEqual(result, want) {
		t.Errorf("Myers fallback: got %v, want %v", result, want)
	}
}

func TestHistogramDiff_BalancedSplit(t *testing.T) {
	// Test that histogram prefers balanced splits
	a := toElements([]string{"a", "b", "anchor", "c", "d"})
	b := toElements([]string{"x", "y", "anchor", "z", "w"})

	got := histogramDiff(a, b, nil)

	// "anchor" should be preserved
	result := applyHistogramDiff(
		[]string{"a", "b", "anchor", "c", "d"},
		[]string{"x", "y", "anchor", "z", "w"},
		got,
	)
	want := []string{"x", "y", "anchor", "z", "w"}

	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestHistogramDiff_LargeInput(t *testing.T) {
	// Test with larger input to exercise the algorithm
	n := 200
	a := make([]string, n)
	b := make([]string, n)

	for i := 0; i < n; i++ {
		a[i] = string(rune('a' + (i % 26)))
		b[i] = string(rune('a' + (i % 26)))
	}

	// Add some unique anchors and changes
	a[50] = "ANCHOR1"
	b[50] = "ANCHOR1"
	a[100] = "ANCHOR2"
	b[100] = "ANCHOR2"
	a[150] = "ANCHOR3"
	b[150] = "ANCHOR3"

	// Make changes between anchors
	b[25] = "CHANGE1"
	b[75] = "CHANGE2"
	b[125] = "CHANGE3"

	ops := DiffHistogram(a, b)
	result := applyDiff(a, b, ops)

	if !reflect.DeepEqual(result, b) {
		t.Error("large input histogram diff failed")
	}
}

func TestHistogramDiff_ProseExample(t *testing.T) {
	// Real-world prose example
	oldText := "The quick brown fox jumps over the lazy dog"
	newText := "A slow red fox leaps over the sleeping cat"

	a := strings.Split(oldText, " ")
	b := strings.Split(newText, " ")

	ops := DiffHistogram(a, b)
	result := applyDiff(a, b, ops)

	if !reflect.DeepEqual(result, b) {
		t.Errorf("prose example: got %v, want %v", result, b)
	}
}

func TestHistogramDiff_CodeTokens(t *testing.T) {
	// Code-like tokens
	a := strings.Split("func main ( ) { fmt . Println ( hello ) }", " ")
	b := strings.Split("func main ( ) { log . Printf ( world ) }", " ")

	ops := DiffHistogram(a, b)
	result := applyDiff(a, b, ops)

	if !reflect.DeepEqual(result, b) {
		t.Errorf("code tokens: got %v, want %v", result, b)
	}
}

// Helper function to apply histogram diff result
func applyHistogramDiff(a, b []string, ops []DiffOp) []string {
	var result []string
	for _, op := range ops {
		switch op.Type {
		case Equal:
			result = append(result, a[op.AStart:op.AEnd]...)
		case Insert:
			result = append(result, b[op.BStart:op.BEnd]...)
		}
	}
	return result
}

// Benchmark histogram diff
func BenchmarkHistogramDiff_Small(b *testing.B) {
	a := toElements([]string{"a", "b", "c", "d", "e"})
	bSeq := toElements([]string{"a", "x", "c", "y", "e"})

	for i := 0; i < b.N; i++ {
		histogramDiff(a, bSeq, nil)
	}
}

func BenchmarkHistogramDiff_Medium(b *testing.B) {
	a := make([]Element, 100)
	bSeq := make([]Element, 100)

	for i := 0; i < 100; i++ {
		a[i] = StringElement(string(rune('a' + (i % 26))))
		bSeq[i] = StringElement(string(rune('a' + (i % 26))))
	}
	bSeq[10] = StringElement("X")
	bSeq[50] = StringElement("Y")
	bSeq[90] = StringElement("Z")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		histogramDiff(a, bSeq, nil)
	}
}

func BenchmarkHistogramDiff_Large(b *testing.B) {
	a := make([]Element, 1000)
	bSeq := make([]Element, 1000)

	for i := 0; i < 1000; i++ {
		a[i] = StringElement(string(rune('a' + (i % 26))))
		bSeq[i] = StringElement(string(rune('a' + (i % 26))))
	}
	for i := 0; i < 100; i++ {
		bSeq[i*10] = StringElement("X")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		histogramDiff(a, bSeq, nil)
	}
}
