package diffx

import (
	"reflect"
	"testing"
)

func TestMergeAdjacentOps(t *testing.T) {
	ops := []DiffOp{
		{Type: Delete, AStart: 0, AEnd: 1, BStart: 0, BEnd: 0},
		{Type: Delete, AStart: 1, AEnd: 2, BStart: 0, BEnd: 0},
		{Type: Equal, AStart: 2, AEnd: 3, BStart: 0, BEnd: 1},
	}

	result := mergeAdjacentOps(ops)

	if len(result) != 2 {
		t.Errorf("Expected 2 ops after merge, got %d: %v", len(result), result)
	}

	if result[0].Type != Delete || result[0].AEnd != 2 {
		t.Errorf("Expected merged delete 0-2, got %v", result[0])
	}
}

func TestMergeAdjacentOps_Empty(t *testing.T) {
	result := mergeAdjacentOps(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}

	result = mergeAdjacentOps([]DiffOp{})
	if len(result) != 0 {
		t.Errorf("expected empty for empty input, got %v", result)
	}
}

func TestMergeAdjacentOps_SingleOp(t *testing.T) {
	ops := []DiffOp{{Type: Equal, AStart: 0, AEnd: 5, BStart: 0, BEnd: 5}}
	result := mergeAdjacentOps(ops)

	if !reflect.DeepEqual(result, ops) {
		t.Errorf("single op should be unchanged, got %v", result)
	}
}

func TestMergeAdjacentOps_NoMerge(t *testing.T) {
	// Different types - no merge
	ops := []DiffOp{
		{Type: Delete, AStart: 0, AEnd: 1, BStart: 0, BEnd: 0},
		{Type: Insert, AStart: 1, AEnd: 1, BStart: 0, BEnd: 1},
		{Type: Equal, AStart: 1, AEnd: 2, BStart: 1, BEnd: 2},
	}

	result := mergeAdjacentOps(ops)

	if len(result) != 3 {
		t.Errorf("expected 3 ops (no merge), got %d: %v", len(result), result)
	}
}

func TestMergeAdjacentOps_Gap(t *testing.T) {
	// Same type but not adjacent - no merge
	ops := []DiffOp{
		{Type: Delete, AStart: 0, AEnd: 2, BStart: 0, BEnd: 0},
		{Type: Delete, AStart: 5, AEnd: 7, BStart: 0, BEnd: 0}, // Gap
	}

	result := mergeAdjacentOps(ops)

	if len(result) != 2 {
		t.Errorf("expected 2 ops (gap prevents merge), got %d: %v", len(result), result)
	}
}

// Tests for boundary shifting
func TestShiftBoundaries_BlankLines(t *testing.T) {
	// Test that blank lines are kept as separators
	a := []string{"line1", "", "line2", "line3", "", "line4"}
	b := []string{"line1", "", "NEW", "line3", "", "line4"}

	ops := Diff(a, b)

	// Apply and verify
	result := applyDiffStrings(a, b, ops)
	if !reflect.DeepEqual(result, b) {
		t.Errorf("Blank line test failed: got %v, want %v", result, b)
	}
}

func TestShiftBoundaries_Punctuation(t *testing.T) {
	// Test that boundaries prefer punctuation marks
	a := []string{"Hello.", "World", "Test"}
	b := []string{"Hello.", "NEW", "Test"}

	// Disable preprocessing for small sequences
	ops := Diff(a, b, WithPreprocessing(false))

	result := applyDiffStrings(a, b, ops)
	if !reflect.DeepEqual(result, b) {
		t.Errorf("Punctuation test failed: got %v, want %v\nOps: %v", result, b, ops)
	}
}

func TestShiftBoundaries_Empty(t *testing.T) {
	result := shiftBoundaries(nil, nil, nil)
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}

	result = shiftBoundaries([]DiffOp{}, nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty for empty input, got %v", result)
	}
}

func TestIsBlank(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", true},
		{"   ", true},
		{"\t", true},
		{"\n", true},
		{"  \t  ", true},
		{"a", false},
		{" a ", false},
		{"hello", false},
	}

	for _, tt := range tests {
		got := isBlank(StringElement(tt.input))
		if got != tt.want {
			t.Errorf("isBlank(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestEndsWithPunctuation(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Hello.", true},
		{"What?", true},
		{"Wow!", true},
		{"Note:", true},
		{"item;", true},
		{"Hello", false},
		{"", false},
		{"   ", false},
	}

	for _, tt := range tests {
		got := endsWithPunctuation(StringElement(tt.input))
		if got != tt.want {
			t.Errorf("endsWithPunctuation(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestStartsWithPunctuation(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"-item", true},
		{"*bullet", true},
		{"#header", true},
		{">quote", true},
		{"Hello", false},
		{"", false},
		{"   ", false},
	}

	for _, tt := range tests {
		got := startsWithPunctuation(StringElement(tt.input))
		if got != tt.want {
			t.Errorf("startsWithPunctuation(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestScoreBoundary(t *testing.T) {
	elems := []Element{
		StringElement("line1"),
		StringElement(""),
		StringElement("line2"),
		StringElement("line3."),
		StringElement("line4"),
	}

	// Score after blank line should be higher
	scoreAfterBlank := scoreBoundary(2, 3, elems)
	scoreNoBlank := scoreBoundary(3, 4, elems)

	if scoreAfterBlank <= scoreNoBlank {
		t.Errorf("Expected higher score after blank line: afterBlank=%d, noBlank=%d",
			scoreAfterBlank, scoreNoBlank)
	}
}

func TestScoreBoundary_Edges(t *testing.T) {
	elems := []Element{
		StringElement("first"),
		StringElement("middle"),
		StringElement("last"),
	}

	// Start of sequence should get bonus
	scoreStart := scoreBoundary(0, 1, elems)
	scoreMid := scoreBoundary(1, 2, elems)

	if scoreStart <= scoreMid {
		t.Errorf("Expected higher score at start: start=%d, mid=%d", scoreStart, scoreMid)
	}

	// End of sequence should get bonus
	scoreEnd := scoreBoundary(2, 3, elems)
	if scoreEnd <= scoreMid {
		t.Errorf("Expected higher score at end: end=%d, mid=%d", scoreEnd, scoreMid)
	}
}

func TestScoreBoundary_Punctuation(t *testing.T) {
	elems := []Element{
		StringElement("sentence."),
		StringElement("next"),
		StringElement("word"),
		StringElement("more"),
	}

	// After punctuation should score higher than middle of sequence
	// Compare index 1 (after "sentence.") vs index 2 (after "next")
	scoreAfterPunct := scoreBoundary(1, 2, elems)
	scoreNoPunct := scoreBoundary(2, 3, elems)

	// Both are in the middle, but one follows punctuation
	if scoreAfterPunct <= scoreNoPunct {
		t.Errorf("Expected higher score after punctuation: afterPunct=%d, noPunct=%d",
			scoreAfterPunct, scoreNoPunct)
	}
}

func TestShiftDelete(t *testing.T) {
	a := toElements([]string{"a", "b", "b", "c"})
	b := toElements([]string{"a", "b", "c"})

	// Delete of "b" at index 1 could shift to index 2 (both are "b")
	op := DiffOp{Type: Delete, AStart: 1, AEnd: 2, BStart: 1, BEnd: 1}
	ops := []DiffOp{op}

	shifted := shiftOp(op, ops, 0, a, b)

	// Should still be a valid delete
	if shifted.Type != Delete {
		t.Errorf("expected Delete, got %v", shifted.Type)
	}
}

func TestShiftInsert(t *testing.T) {
	a := toElements([]string{"a", "c"})
	b := toElements([]string{"a", "b", "b", "c"})

	// Insert of "b" could potentially shift
	op := DiffOp{Type: Insert, AStart: 1, AEnd: 1, BStart: 1, BEnd: 2}
	ops := []DiffOp{op}

	shifted := shiftOp(op, ops, 0, a, b)

	// Should still be a valid insert
	if shifted.Type != Insert {
		t.Errorf("expected Insert, got %v", shifted.Type)
	}
}

// Helper to apply diff (duplicated here to avoid import cycle)
func applyDiffStrings(a, b []string, ops []DiffOp) []string {
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
