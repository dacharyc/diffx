package diffx

import "testing"

func TestIsqrt(t *testing.T) {
	tests := []struct {
		n    int
		want int
	}{
		{0, 0},
		{1, 1},
		{4, 2},
		{9, 3},
		{10, 3},
		{15, 3},
		{16, 4},
		{100, 10},
		{101, 10},
		{10000, 100},
	}

	for _, tt := range tests {
		got := isqrt(tt.n)
		if got != tt.want {
			t.Errorf("isqrt(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestIsqrt_Negative(t *testing.T) {
	// Negative input should return 0 (or handle gracefully)
	got := isqrt(-1)
	if got != 0 {
		t.Errorf("isqrt(-1) = %d, want 0", got)
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		x    int
		want int
	}{
		{0, 0},
		{5, 5},
		{-5, 5},
		{-100, 100},
		{100, 100},
		{-1, 1},
		{1, 1},
	}

	for _, tt := range tests {
		got := abs(tt.x)
		if got != tt.want {
			t.Errorf("abs(%d) = %d, want %d", tt.x, got, tt.want)
		}
	}
}

func TestFindMiddleSnake_Empty(t *testing.T) {
	// Test with empty sequences
	o := defaultOptions()
	ctx := newDiffContext([]Element{}, []Element{}, o)

	// This should not panic
	part := ctx.findMiddleSnake(0, 0, 0, 0, false)

	if part.xmid != 0 || part.ymid != 0 {
		t.Errorf("expected (0,0) for empty, got (%d,%d)", part.xmid, part.ymid)
	}
}

func TestFindMiddleSnake_Equal(t *testing.T) {
	a := toElements([]string{"a", "b", "c"})
	b := toElements([]string{"a", "b", "c"})

	o := defaultOptions()
	ctx := newDiffContext(a, b, o)

	part := ctx.findMiddleSnake(0, 3, 0, 3, false)

	// For equal sequences, should find a path through
	if part.xmid < 0 || part.ymid < 0 {
		t.Errorf("invalid partition for equal sequences: %+v", part)
	}
}

func TestFindMiddleSnake_AllDifferent(t *testing.T) {
	a := toElements([]string{"a", "b", "c"})
	b := toElements([]string{"x", "y", "z"})

	o := defaultOptions()
	ctx := newDiffContext(a, b, o)

	part := ctx.findMiddleSnake(0, 3, 0, 3, false)

	// Should find some partition
	if part.xmid < 0 || part.ymid < 0 {
		t.Errorf("invalid partition for different sequences: %+v", part)
	}
}

func TestFindMiddleSnake_WithHeuristics(t *testing.T) {
	// Create a larger sequence where heuristics might kick in
	n := 100
	a := make([]Element, n)
	b := make([]Element, n)

	for i := 0; i < n; i++ {
		a[i] = StringElement(string(rune('a' + (i % 26))))
		b[i] = StringElement(string(rune('z' - (i % 26)))) // Different
	}

	o := defaultOptions()
	o.useHeuristic = true
	ctx := newDiffContext(a, b, o)

	part := ctx.findMiddleSnake(0, n, 0, n, false)

	// Should find some partition
	if part.xmid < 0 || part.xmid > n || part.ymid < 0 || part.ymid > n {
		t.Errorf("invalid partition with heuristics: %+v", part)
	}
}

// Benchmark snake finding
func BenchmarkFindMiddleSnake_Small(b *testing.B) {
	a := toElements([]string{"a", "b", "c", "d", "e"})
	bSeq := toElements([]string{"a", "x", "c", "y", "e"})

	o := defaultOptions()
	ctx := newDiffContext(a, bSeq, o)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.findMiddleSnake(0, 5, 0, 5, false)
	}
}

func BenchmarkFindMiddleSnake_Large(b *testing.B) {
	n := 500
	a := make([]Element, n)
	bSeq := make([]Element, n)

	for i := 0; i < n; i++ {
		a[i] = StringElement(string(rune('a' + (i % 26))))
		bSeq[i] = StringElement(string(rune('a' + ((i + 1) % 26))))
	}

	o := defaultOptions()
	ctx := newDiffContext(a, bSeq, o)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.findMiddleSnake(0, n, 0, n, false)
	}
}
