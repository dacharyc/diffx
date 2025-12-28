package diffx

import (
	"reflect"
	"testing"
)

func TestEliminateWeakAnchors_Empty(t *testing.T) {
	// Empty and single-element cases should return as-is
	tests := []struct {
		name string
		ops  []DiffOp
		want []DiffOp
	}{
		{
			name: "nil input",
			ops:  nil,
			want: nil,
		},
		{
			name: "empty slice",
			ops:  []DiffOp{},
			want: []DiffOp{},
		},
		{
			name: "single op",
			ops:  []DiffOp{{Type: Equal, AStart: 0, AEnd: 5, BStart: 0, BEnd: 5}},
			want: []DiffOp{{Type: Equal, AStart: 0, AEnd: 5, BStart: 0, BEnd: 5}},
		},
	}

	a := []Element{StringElement("a")}
	b := []Element{StringElement("b")}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eliminateWeakAnchors(tt.ops, a, b)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("eliminateWeakAnchors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEliminateWeakAnchors_MergesAdjacent(t *testing.T) {
	// Should merge adjacent operations of the same type
	ops := []DiffOp{
		{Type: Delete, AStart: 0, AEnd: 1, BStart: 0, BEnd: 0},
		{Type: Delete, AStart: 1, AEnd: 2, BStart: 0, BEnd: 0},
		{Type: Delete, AStart: 2, AEnd: 3, BStart: 0, BEnd: 0},
	}

	a := toElements([]string{"a", "b", "c"})
	b := []Element{}

	got := eliminateWeakAnchors(ops, a, b)

	if len(got) != 1 {
		t.Fatalf("expected 1 merged op, got %d: %v", len(got), got)
	}

	if got[0].Type != Delete || got[0].AStart != 0 || got[0].AEnd != 3 {
		t.Errorf("expected merged Delete 0-3, got %v", got[0])
	}
}

func TestEliminateWeakAnchors_PreservesDifferentTypes(t *testing.T) {
	// Should not merge operations of different types
	ops := []DiffOp{
		{Type: Delete, AStart: 0, AEnd: 1, BStart: 0, BEnd: 0},
		{Type: Insert, AStart: 1, AEnd: 1, BStart: 0, BEnd: 1},
		{Type: Equal, AStart: 1, AEnd: 2, BStart: 1, BEnd: 2},
	}

	a := toElements([]string{"a", "b"})
	b := toElements([]string{"x", "b"})

	got := eliminateWeakAnchors(ops, a, b)

	if len(got) != 3 {
		t.Errorf("expected 3 ops preserved, got %d: %v", len(got), got)
	}
}

func TestEliminateWeakAnchors_MergesNonAdjacent(t *testing.T) {
	// Adjacent same-type ops should merge, others preserved
	ops := []DiffOp{
		{Type: Equal, AStart: 0, AEnd: 1, BStart: 0, BEnd: 1},
		{Type: Delete, AStart: 1, AEnd: 2, BStart: 1, BEnd: 1},
		{Type: Delete, AStart: 2, AEnd: 3, BStart: 1, BEnd: 1},
		{Type: Insert, AStart: 3, AEnd: 3, BStart: 1, BEnd: 2},
		{Type: Equal, AStart: 3, AEnd: 4, BStart: 2, BEnd: 3},
	}

	a := toElements([]string{"a", "b", "c", "d"})
	b := toElements([]string{"a", "x", "d"})

	got := eliminateWeakAnchors(ops, a, b)

	// Should have: Equal, merged Delete, Insert, Equal = 4 ops
	if len(got) != 4 {
		t.Errorf("expected 4 ops after merge, got %d: %v", len(got), got)
	}

	// Check the merged delete
	if got[1].Type != Delete || got[1].AStart != 1 || got[1].AEnd != 3 {
		t.Errorf("expected merged Delete 1-3, got %v", got[1])
	}
}

func TestWithAnchorElimination(t *testing.T) {
	a := []string{"the", "quick", "fox"}
	b := []string{"a", "slow", "fox"}

	// Test with anchor elimination enabled (default)
	ops1 := Diff(a, b, WithAnchorElimination(true))
	result1 := applyDiff(a, b, ops1)
	if !reflect.DeepEqual(result1, b) {
		t.Errorf("with anchor elimination: got %v, want %v", result1, b)
	}

	// Test with anchor elimination disabled
	ops2 := Diff(a, b, WithAnchorElimination(false))
	result2 := applyDiff(a, b, ops2)
	if !reflect.DeepEqual(result2, b) {
		t.Errorf("without anchor elimination: got %v, want %v", result2, b)
	}
}
