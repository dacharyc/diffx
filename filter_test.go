package diffx

import (
	"reflect"
	"testing"
)

func TestFilterConfusingElements_Empty(t *testing.T) {
	tests := []struct {
		name string
		a, b []Element
	}{
		{"both empty", []Element{}, []Element{}},
		{"a empty", []Element{}, toElements([]string{"x"})},
		{"b empty", toElements([]string{"x"}), []Element{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotA, gotB, mapping := filterConfusingElements(tt.a, tt.b)

			// Should return original sequences unchanged
			if !reflect.DeepEqual(gotA, tt.a) || !reflect.DeepEqual(gotB, tt.b) {
				t.Errorf("expected unchanged sequences")
			}
			if mapping != nil {
				t.Error("expected nil mapping for empty input")
			}
		})
	}
}

func TestFilterConfusingElements_NoHighFrequency(t *testing.T) {
	// All elements are unique - no filtering needed
	a := toElements([]string{"unique1", "unique2", "unique3"})
	b := toElements([]string{"unique1", "unique4", "unique3"})

	gotA, gotB, mapping := filterConfusingElements(a, b)

	// Should return original sequences (most elements kept)
	if mapping != nil {
		// If filtering happened, verify it still works correctly
		if len(gotA) == 0 && len(gotB) == 0 {
			t.Error("should not filter everything out")
		}
	}
}

func TestFilterConfusingElements_HighFrequency(t *testing.T) {
	// Create sequences with high-frequency elements
	// Need enough repetition to trigger filtering
	a := make([]Element, 100)
	b := make([]Element, 100)

	// Fill with high-frequency element
	for i := 0; i < 100; i++ {
		a[i] = StringElement("common")
		b[i] = StringElement("common")
	}

	// Add some unique elements
	a[50] = StringElement("uniqueA")
	b[50] = StringElement("uniqueB")

	filteredA, filteredB, mapping := filterConfusingElements(a, b)

	// The unique elements should be kept, high-frequency may be filtered
	if mapping == nil {
		// No filtering happened - that's okay if threshold wasn't met
		return
	}

	// Verify mapping is valid
	if len(mapping.aToOrig) > len(a) {
		t.Error("mapping has more entries than original")
	}

	// Verify filtered sequences are not larger than originals
	if len(filteredA) > len(a) || len(filteredB) > len(b) {
		t.Error("filtered sequences should not be larger")
	}
}

func TestFilterSequence_KeepOnly(t *testing.T) {
	elems := toElements([]string{"a", "b", "c", "d"})
	classes := []elementClass{keep, keep, keep, keep}

	result, toOrig := filterSequence(elems, classes)

	if len(result) != 4 {
		t.Errorf("expected 4 elements, got %d", len(result))
	}

	for i, idx := range toOrig {
		if idx != i {
			t.Errorf("expected toOrig[%d] = %d, got %d", i, i, idx)
		}
	}
}

func TestFilterSequence_DiscardOnly(t *testing.T) {
	elems := toElements([]string{"a", "b", "c", "d"})
	classes := []elementClass{discard, discard, discard, discard}

	result, toOrig := filterSequence(elems, classes)

	if len(result) != 0 {
		t.Errorf("expected 0 elements, got %d", len(result))
	}
	if len(toOrig) != 0 {
		t.Errorf("expected empty toOrig, got %d entries", len(toOrig))
	}
}

func TestFilterSequence_Mixed(t *testing.T) {
	elems := toElements([]string{"a", "b", "c", "d", "e"})
	classes := []elementClass{keep, discard, keep, discard, keep}

	result, toOrig := filterSequence(elems, classes)

	if len(result) != 3 {
		t.Errorf("expected 3 elements, got %d", len(result))
	}

	expectedOrig := []int{0, 2, 4}
	if !reflect.DeepEqual(toOrig, expectedOrig) {
		t.Errorf("expected toOrig %v, got %v", expectedOrig, toOrig)
	}
}

func TestFilterSequence_Provisional(t *testing.T) {
	elems := toElements([]string{"keep1", "prov", "keep2", "prov2", "discard"})
	classes := []elementClass{keep, provisional, keep, provisional, discard}

	result, _ := filterSequence(elems, classes)

	// Provisional elements adjacent to keep should be kept
	// prov (index 1) is between keep1 and keep2 - should be kept
	// prov2 (index 3) is between keep2 and discard - has keep before, should be kept

	if len(result) < 3 {
		t.Errorf("expected at least 3 elements (keeps + some provisionals), got %d", len(result))
	}
}

func TestFilterSequence_ProvisionalAtBoundary(t *testing.T) {
	elems := toElements([]string{"prov", "keep", "prov2"})
	classes := []elementClass{provisional, keep, provisional}

	result, _ := filterSequence(elems, classes)

	// prov at start: next is keep -> should be kept
	// prov2 at end: prev is keep -> should be kept
	if len(result) != 3 {
		t.Errorf("expected 3 elements (provisionals adjacent to keep), got %d", len(result))
	}
}

func TestMergeOps_Empty(t *testing.T) {
	result := mergeOps(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}

	result = mergeOps([]DiffOp{})
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestMergeOps_SingleOp(t *testing.T) {
	ops := []DiffOp{{Type: Equal, AStart: 0, AEnd: 5, BStart: 0, BEnd: 5}}
	result := mergeOps(ops)

	if !reflect.DeepEqual(result, ops) {
		t.Errorf("single op should be unchanged")
	}
}

func TestMergeOps_AdjacentSameType(t *testing.T) {
	ops := []DiffOp{
		{Type: Delete, AStart: 0, AEnd: 2, BStart: 0, BEnd: 0},
		{Type: Delete, AStart: 2, AEnd: 4, BStart: 0, BEnd: 0},
		{Type: Delete, AStart: 4, AEnd: 6, BStart: 0, BEnd: 0},
	}

	result := mergeOps(ops)

	if len(result) != 1 {
		t.Fatalf("expected 1 merged op, got %d", len(result))
	}

	if result[0].AStart != 0 || result[0].AEnd != 6 {
		t.Errorf("expected merged range 0-6, got %d-%d", result[0].AStart, result[0].AEnd)
	}
}

func TestMergeOps_NonAdjacentSameType(t *testing.T) {
	ops := []DiffOp{
		{Type: Delete, AStart: 0, AEnd: 2, BStart: 0, BEnd: 0},
		{Type: Delete, AStart: 5, AEnd: 7, BStart: 0, BEnd: 0}, // Gap at 2-5
	}

	result := mergeOps(ops)

	// Should not merge due to gap
	if len(result) != 2 {
		t.Errorf("expected 2 separate ops (non-adjacent), got %d", len(result))
	}
}

func TestMergeOps_DifferentTypes(t *testing.T) {
	ops := []DiffOp{
		{Type: Delete, AStart: 0, AEnd: 2, BStart: 0, BEnd: 0},
		{Type: Insert, AStart: 2, AEnd: 2, BStart: 0, BEnd: 2},
		{Type: Equal, AStart: 2, AEnd: 5, BStart: 2, BEnd: 5},
	}

	result := mergeOps(ops)

	if len(result) != 3 {
		t.Errorf("expected 3 ops (different types), got %d", len(result))
	}
}

func TestIndexMapping_MapOps_Nil(t *testing.T) {
	ops := []DiffOp{{Type: Equal, AStart: 0, AEnd: 5, BStart: 0, BEnd: 5}}

	var m *indexMapping = nil
	result := m.mapOps(ops)

	if !reflect.DeepEqual(result, ops) {
		t.Error("nil mapping should return ops unchanged")
	}
}

func TestIndexMapping_MapOps_Simple(t *testing.T) {
	// Simulate filtering where elements at indices 1 and 3 were removed
	// Original: [a, X, b, Y, c] -> Filtered: [a, b, c]
	m := &indexMapping{
		aToOrig: []int{0, 2, 4}, // filtered indices map to original indices
		bToOrig: []int{0, 2, 4},
		origN:   5,
		origM:   5,
	}

	// Op on filtered sequence: Equal for all 3 elements
	ops := []DiffOp{
		{Type: Equal, AStart: 0, AEnd: 3, BStart: 0, BEnd: 3},
	}

	result := m.mapOps(ops)

	// Should expand to cover gaps
	// After mapping and merging, we should have operations that cover the original indices
	// and include the filtered elements as deletes/inserts

	// Verify total coverage
	totalA := 0
	for _, op := range result {
		if op.Type == Equal || op.Type == Delete {
			totalA += op.AEnd - op.AStart
		}
	}

	if totalA != 5 {
		t.Errorf("expected total A coverage of 5, got %d (ops: %v)", totalA, result)
	}
}

func TestIndexMapping_MapOps_WithChanges(t *testing.T) {
	// Filtered sequence has a delete
	// Original: [a, X, b, Y, c]
	// Filtered: [a, b, c]
	// Op: delete b (filtered index 1)
	m := &indexMapping{
		aToOrig: []int{0, 2, 4},
		bToOrig: []int{0, 2, 4},
		origN:   5,
		origM:   5,
	}

	ops := []DiffOp{
		{Type: Equal, AStart: 0, AEnd: 1, BStart: 0, BEnd: 1},
		{Type: Delete, AStart: 1, AEnd: 2, BStart: 1, BEnd: 1},
		{Type: Equal, AStart: 2, AEnd: 3, BStart: 1, BEnd: 2},
	}

	result := m.mapOps(ops)

	// Verify the result is valid
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}

	// Check that all original indices are covered
	aCovered := make([]bool, 5)
	for _, op := range result {
		if op.Type == Equal || op.Type == Delete {
			for i := op.AStart; i < op.AEnd; i++ {
				if i < 5 {
					aCovered[i] = true
				}
			}
		}
	}

	for i, covered := range aCovered {
		if !covered {
			t.Errorf("original index %d not covered", i)
		}
	}
}

func TestFilterConfusingElements_Integration(t *testing.T) {
	// Integration test: filter, diff, map back
	a := toElements([]string{"the", "quick", "fox", "the", "end"})
	b := toElements([]string{"the", "slow", "fox", "the", "end"})

	filteredA, filteredB, mapping := filterConfusingElements(a, b)

	if mapping == nil {
		// No filtering - sequences were similar enough
		// Just verify the original sequences work with Diff
		ops := Diff([]string{"the", "quick", "fox", "the", "end"},
			[]string{"the", "slow", "fox", "the", "end"})
		result := applyDiff(
			[]string{"the", "quick", "fox", "the", "end"},
			[]string{"the", "slow", "fox", "the", "end"},
			ops,
		)
		want := []string{"the", "slow", "fox", "the", "end"}
		if !reflect.DeepEqual(result, want) {
			t.Errorf("got %v, want %v", result, want)
		}
		return
	}

	// If filtering happened, verify the filtered sequences are smaller or equal
	if len(filteredA) > len(a) || len(filteredB) > len(b) {
		t.Error("filtered sequences should not be larger than originals")
	}
}

// Benchmark filtering
func BenchmarkFilterConfusingElements_Small(b *testing.B) {
	a := toElements([]string{"the", "quick", "brown", "fox", "jumps"})
	bSeq := toElements([]string{"a", "slow", "red", "fox", "leaps"})

	for i := 0; i < b.N; i++ {
		filterConfusingElements(a, bSeq)
	}
}

func BenchmarkFilterConfusingElements_Large(b *testing.B) {
	a := make([]Element, 1000)
	bSeq := make([]Element, 1000)

	words := []string{"the", "a", "an", "in", "on", "for", "with", "unique1", "unique2", "unique3"}
	for i := 0; i < 1000; i++ {
		a[i] = StringElement(words[i%len(words)])
		bSeq[i] = StringElement(words[(i+1)%len(words)])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filterConfusingElements(a, bSeq)
	}
}
