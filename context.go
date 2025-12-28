package diffx

import "math"

// partition holds the result from findMiddleSnake().
// It represents the midpoint where the edit path can be split.
type partition struct {
	xmid, ymid int  // midpoint coordinates in the edit graph
	loMinimal  bool // whether lower half needs minimal search
	hiMinimal  bool // whether upper half needs minimal search
}

// diffContext holds algorithm state during comparison.
type diffContext struct {
	xvec, yvec   []Element // sequences being compared
	fdiag, bdiag []int     // forward/backward diagonal arrays
	xchanges     []bool    // marks changed elements in xvec
	ychanges     []bool    // marks changed elements in yvec
	useHeuristic bool      // enable speed heuristics
	costLimit    int       // max cost before early termination
}

// newDiffContext creates a new context for comparing two sequences.
func newDiffContext(a, b []Element, opts *options) *diffContext {
	n := len(a)
	m := len(b)

	// Diagonal array size: we need diagonals from -(m) to +(n)
	// The array is indexed by [k + offset] where offset = m
	diagSize := n + m + 3

	ctx := &diffContext{
		xvec:         a,
		yvec:         b,
		fdiag:        make([]int, diagSize),
		bdiag:        make([]int, diagSize),
		xchanges:     make([]bool, n),
		ychanges:     make([]bool, m),
		useHeuristic: opts.useHeuristic,
		costLimit:    opts.costLimit,
	}

	// Auto-calculate cost limit if not specified
	if ctx.costLimit == 0 && ctx.useHeuristic {
		// sqrt(n) * sqrt(m) / 4, but at least 256
		ctx.costLimit = int(math.Sqrt(float64(n)) * math.Sqrt(float64(m)) / 4)
		if ctx.costLimit < 256 {
			ctx.costLimit = 256
		}
	}

	return ctx
}

// diagOffset returns the offset to use when indexing diagonal arrays.
// Diagonal k is stored at index k + offset.
func (ctx *diffContext) diagOffset() int {
	return len(ctx.yvec) + 1
}

// markDeleted marks elements in xvec[xoff:xlim] as deleted.
func (ctx *diffContext) markDeleted(xoff, xlim int) {
	for i := xoff; i < xlim; i++ {
		ctx.xchanges[i] = true
	}
}

// markInserted marks elements in yvec[yoff:ylim] as inserted.
func (ctx *diffContext) markInserted(yoff, ylim int) {
	for i := yoff; i < ylim; i++ {
		ctx.ychanges[i] = true
	}
}

// equal reports whether xvec[i] equals yvec[j].
func (ctx *diffContext) equal(i, j int) bool {
	return ctx.xvec[i].Equal(ctx.yvec[j])
}
