package diffx

import (
	"reflect"
	"testing"
)

func TestDiff_Empty(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want []DiffOp
	}{
		{
			name: "both empty",
			a:    []string{},
			b:    []string{},
			want: nil,
		},
		{
			name: "a empty",
			a:    []string{},
			b:    []string{"x", "y"},
			want: []DiffOp{
				{Type: Insert, AStart: 0, AEnd: 0, BStart: 0, BEnd: 2},
			},
		},
		{
			name: "b empty",
			a:    []string{"x", "y"},
			b:    []string{},
			want: []DiffOp{
				{Type: Delete, AStart: 0, AEnd: 2, BStart: 0, BEnd: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Diff(tt.a, tt.b, WithPreprocessing(false), WithPostprocessing(false))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Diff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiff_Equal(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"a", "b", "c"}

	got := Diff(a, b, WithPreprocessing(false), WithPostprocessing(false))
	want := []DiffOp{
		{Type: Equal, AStart: 0, AEnd: 3, BStart: 0, BEnd: 3},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Diff() = %v, want %v", got, want)
	}
}

func TestDiff_AllDifferent(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"x", "y", "z"}

	got := Diff(a, b, WithPreprocessing(false), WithPostprocessing(false))

	// Should have delete and insert operations
	hasDelete := false
	hasInsert := false
	for _, op := range got {
		if op.Type == Delete {
			hasDelete = true
		}
		if op.Type == Insert {
			hasInsert = true
		}
	}

	if !hasDelete || !hasInsert {
		t.Errorf("Expected delete and insert operations, got %v", got)
	}
}

func TestDiff_SimpleChange(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"a", "x", "c"}

	got := Diff(a, b, WithPreprocessing(false), WithPostprocessing(false))

	// Verify the result:
	// - Equal: a at position 0
	// - Delete: b at position 1
	// - Insert: x at position 1
	// - Equal: c at position 2

	if len(got) < 3 {
		t.Fatalf("Expected at least 3 operations, got %d: %v", len(got), got)
	}

	// First should be equal for "a"
	if got[0].Type != Equal || got[0].AEnd-got[0].AStart != 1 {
		t.Errorf("Expected equal op for 'a', got %v", got[0])
	}
}

func TestDiff_Insert(t *testing.T) {
	a := []string{"a", "c"}
	b := []string{"a", "b", "c"}

	got := Diff(a, b, WithPreprocessing(false), WithPostprocessing(false))

	// Should have:
	// - Equal: a
	// - Insert: b
	// - Equal: c

	foundInsert := false
	for _, op := range got {
		if op.Type == Insert {
			foundInsert = true
			if op.BEnd-op.BStart != 1 {
				t.Errorf("Expected insert of 1 element, got %v", op)
			}
		}
	}

	if !foundInsert {
		t.Errorf("Expected insert operation, got %v", got)
	}
}

func TestDiff_Delete(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"a", "c"}

	got := Diff(a, b, WithPreprocessing(false), WithPostprocessing(false))

	// Should have:
	// - Equal: a
	// - Delete: b
	// - Equal: c

	foundDelete := false
	for _, op := range got {
		if op.Type == Delete {
			foundDelete = true
			if op.AEnd-op.AStart != 1 {
				t.Errorf("Expected delete of 1 element, got %v", op)
			}
		}
	}

	if !foundDelete {
		t.Errorf("Expected delete operation, got %v", got)
	}
}

func TestDiff_ApplyProducesB(t *testing.T) {
	// Property test: applying the diff to A should produce B
	tests := []struct {
		name string
		a, b []string
	}{
		{"simple", []string{"a", "b", "c"}, []string{"a", "x", "c"}},
		{"insert", []string{"a", "c"}, []string{"a", "b", "c"}},
		{"delete", []string{"a", "b", "c"}, []string{"a", "c"}},
		{"replace all", []string{"a", "b"}, []string{"x", "y"}},
		{"complex", []string{"a", "b", "c", "d", "e"}, []string{"a", "x", "c", "y", "e"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops := Diff(tt.a, tt.b, WithPreprocessing(false), WithPostprocessing(false))
			result := applyDiff(tt.a, tt.b, ops)

			if !reflect.DeepEqual(result, tt.b) {
				t.Errorf("Applying diff to %v did not produce %v, got %v\nOps: %v",
					tt.a, tt.b, result, ops)
			}
		})
	}
}

// applyDiff applies a diff to sequence a to produce b
func applyDiff(a, b []string, ops []DiffOp) []string {
	var result []string

	for _, op := range ops {
		switch op.Type {
		case Equal:
			result = append(result, a[op.AStart:op.AEnd]...)
		case Delete:
			// Don't add deleted elements
		case Insert:
			result = append(result, b[op.BStart:op.BEnd]...)
		}
	}

	return result
}

func TestOpType_String(t *testing.T) {
	tests := []struct {
		op   OpType
		want string
	}{
		{Equal, "Equal"},
		{Insert, "Insert"},
		{Delete, "Delete"},
		{OpType(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("OpType(%d).String() = %q, want %q", tt.op, got, tt.want)
		}
	}
}

func TestDiff_WithOptions(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"a", "x", "c"}

	// Test with heuristics disabled
	ops1 := Diff(a, b, WithHeuristic(false))
	if len(ops1) == 0 {
		t.Error("Expected non-empty diff with heuristics disabled")
	}

	// Test with minimal forced
	ops2 := Diff(a, b, WithMinimal(true))
	if len(ops2) == 0 {
		t.Error("Expected non-empty diff with minimal forced")
	}
}

func TestDiff_LargerSequences(t *testing.T) {
	// Test with larger sequences to exercise the algorithm more
	a := make([]string, 100)
	b := make([]string, 100)

	for i := 0; i < 100; i++ {
		a[i] = string(rune('a' + (i % 26)))
		b[i] = string(rune('a' + (i % 26)))
	}

	// Modify some elements in b
	b[10] = "X"
	b[50] = "Y"
	b[90] = "Z"

	ops := Diff(a, b, WithPreprocessing(false), WithPostprocessing(false))

	// Verify by applying
	result := applyDiff(a, b, ops)
	if !reflect.DeepEqual(result, b) {
		t.Errorf("Failed to produce correct result for larger sequences")
	}
}

func TestDiff_FoxExample(t *testing.T) {
	// The motivating example from the plan
	old := []string{"The", "quick", "brown", "fox", "jumps"}
	new := []string{"A", "slow", "red", "fox", "leaps"}

	ops := Diff(old, new, WithPreprocessing(false), WithPostprocessing(false))

	// Verify the diff produces correct output
	result := applyDiff(old, new, ops)
	if !reflect.DeepEqual(result, new) {
		t.Errorf("Fox example failed: got %v, want %v", result, new)
	}

	// Check that "fox" is preserved (appears in an Equal operation)
	foxPreserved := false
	for _, op := range ops {
		if op.Type == Equal {
			for i := op.AStart; i < op.AEnd; i++ {
				if old[i] == "fox" {
					foxPreserved = true
				}
			}
		}
	}

	if !foxPreserved {
		t.Error("Expected 'fox' to be preserved in an Equal operation")
	}
}

// Tests for heuristics
func TestDiff_HeuristicsVsMinimal(t *testing.T) {
	// Generate a sequence where heuristics might give different results
	a := make([]string, 200)
	b := make([]string, 200)

	for i := 0; i < 200; i++ {
		a[i] = string(rune('a' + (i % 26)))
		b[i] = string(rune('a' + (i % 26)))
	}

	// Scatter many changes
	for i := 0; i < 50; i++ {
		b[i*4] = "X"
	}

	// Both should produce correct results
	opsHeuristic := Diff(a, b, WithHeuristic(true), WithPreprocessing(false))
	opsMinimal := Diff(a, b, WithMinimal(true), WithPreprocessing(false))

	resultH := applyDiff(a, b, opsHeuristic)
	resultM := applyDiff(a, b, opsMinimal)

	if !reflect.DeepEqual(resultH, b) {
		t.Error("Heuristic diff did not produce correct result")
	}
	if !reflect.DeepEqual(resultM, b) {
		t.Error("Minimal diff did not produce correct result")
	}
}

func TestDiff_LargeWithHeuristics(t *testing.T) {
	// Test with a large sequence to exercise cost limit heuristics
	n := 500
	a := make([]string, n)
	b := make([]string, n)

	for i := 0; i < n; i++ {
		a[i] = string(rune('a' + (i % 26)))
		b[i] = string(rune('a' + ((i + 1) % 26))) // Shifted pattern
	}

	// Keep some anchors
	for i := 0; i < n; i += 50 {
		b[i] = a[i]
	}

	ops := Diff(a, b, WithHeuristic(true))
	result := applyDiff(a, b, ops)

	if !reflect.DeepEqual(result, b) {
		t.Error("Large heuristic diff did not produce correct result")
	}
}

func TestDiff_PathologicalCase(t *testing.T) {
	// All elements are the same - tests the algorithm's behavior
	// when there are many possible alignments
	a := make([]string, 50)
	b := make([]string, 50)

	for i := 0; i < 50; i++ {
		a[i] = "x"
		b[i] = "x"
	}

	// Change a few in the middle
	b[25] = "y"

	ops := Diff(a, b)
	result := applyDiff(a, b, ops)

	if !reflect.DeepEqual(result, b) {
		t.Errorf("Pathological case failed: got %v, want %v", result, b)
	}
}

func TestDiff_WithCostLimit(t *testing.T) {
	a := make([]string, 100)
	b := make([]string, 100)

	for i := 0; i < 100; i++ {
		a[i] = string(rune('a' + (i % 26)))
		b[i] = string(rune('z' - (i % 26))) // Very different
	}

	// With a low cost limit, should still produce valid (if not minimal) result
	ops := Diff(a, b, WithCostLimit(10))
	result := applyDiff(a, b, ops)

	if !reflect.DeepEqual(result, b) {
		t.Error("Cost limit diff did not produce correct result")
	}
}

// Benchmark tests
func BenchmarkDiff_Small(b *testing.B) {
	a := []string{"a", "b", "c", "d", "e"}
	bSeq := []string{"a", "x", "c", "y", "e"}

	for i := 0; i < b.N; i++ {
		Diff(a, bSeq)
	}
}

func BenchmarkDiff_Medium(b *testing.B) {
	a := make([]string, 100)
	bSeq := make([]string, 100)

	for i := 0; i < 100; i++ {
		a[i] = string(rune('a' + (i % 26)))
		bSeq[i] = string(rune('a' + (i % 26)))
	}
	bSeq[10] = "X"
	bSeq[50] = "Y"
	bSeq[90] = "Z"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Diff(a, bSeq)
	}
}

func BenchmarkDiff_Large(b *testing.B) {
	a := make([]string, 1000)
	bSeq := make([]string, 1000)

	for i := 0; i < 1000; i++ {
		a[i] = string(rune('a' + (i % 26)))
		bSeq[i] = string(rune('a' + (i % 26)))
	}
	// Scatter changes
	for i := 0; i < 100; i++ {
		bSeq[i*10] = "X"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Diff(a, bSeq)
	}
}
