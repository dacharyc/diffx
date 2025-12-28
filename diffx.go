// Package diffx implements the Myers O(ND) diff algorithm with heuristics
// for better output quality on large files with many small, scattered changes.
//
// Unlike simple Myers implementations, diffx includes:
//   - Preprocessing: Filters out high-frequency elements that cause spurious matches
//   - Heuristics: Early termination for expensive comparisons
//   - Postprocessing: Shifts diff boundaries for more readable output
package diffx

// OpType identifies the type of edit operation.
type OpType int

const (
	// Equal means the elements are unchanged.
	Equal OpType = iota
	// Insert means elements were added to B that are not in A.
	Insert
	// Delete means elements were removed from A that are not in B.
	Delete
)

// String returns a string representation of the OpType.
func (t OpType) String() string {
	switch t {
	case Equal:
		return "Equal"
	case Insert:
		return "Insert"
	case Delete:
		return "Delete"
	default:
		return "Unknown"
	}
}

// DiffOp represents a single edit operation with index ranges.
type DiffOp struct {
	Type   OpType
	AStart int // start index in sequence A (inclusive)
	AEnd   int // end index in sequence A (exclusive)
	BStart int // start index in sequence B (inclusive)
	BEnd   int // end index in sequence B (exclusive)
}

// options holds configuration for the diff algorithm.
type options struct {
	useHeuristic   bool
	forceMinimal   bool
	costLimit      int
	preprocessing  bool
	postprocessing bool
}

// defaultOptions returns options with sensible defaults.
func defaultOptions() *options {
	return &options{
		useHeuristic:   true,
		forceMinimal:   false,
		costLimit:      0, // auto-calculated
		preprocessing:  true,
		postprocessing: true,
	}
}

// Option configures diff behavior.
type Option func(*options)

// WithHeuristic enables or disables speed heuristics.
// Default: true.
func WithHeuristic(enabled bool) Option {
	return func(o *options) {
		o.useHeuristic = enabled
	}
}

// WithMinimal forces minimal edit script even if slow.
// Default: false.
func WithMinimal(minimal bool) Option {
	return func(o *options) {
		o.forceMinimal = minimal
		if minimal {
			o.useHeuristic = false
		}
	}
}

// WithCostLimit sets custom early termination threshold.
// 0 means auto-calculate based on input size.
// Default: 0.
func WithCostLimit(n int) Option {
	return func(o *options) {
		o.costLimit = n
	}
}

// WithPreprocessing enables or disables confusing element filtering.
// Default: true.
func WithPreprocessing(enabled bool) Option {
	return func(o *options) {
		o.preprocessing = enabled
	}
}

// WithPostprocessing enables or disables boundary shifting.
// Default: true.
func WithPostprocessing(enabled bool) Option {
	return func(o *options) {
		o.postprocessing = enabled
	}
}

// Diff compares two string slices and returns edit operations.
func Diff(a, b []string, opts ...Option) []DiffOp {
	return DiffElements(toElements(a), toElements(b), opts...)
}

// DiffElements compares arbitrary Element slices.
func DiffElements(a, b []Element, opts ...Option) []DiffOp {
	// Apply options
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	// Handle trivial cases
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	if len(a) == 0 {
		return []DiffOp{{
			Type:   Insert,
			AStart: 0,
			AEnd:   0,
			BStart: 0,
			BEnd:   len(b),
		}}
	}
	if len(b) == 0 {
		return []DiffOp{{
			Type:   Delete,
			AStart: 0,
			AEnd:   len(a),
			BStart: 0,
			BEnd:   0,
		}}
	}

	// Create context and run algorithm
	ctx := newDiffContext(a, b, o)

	// Preprocessing: filter confusing elements
	var mapping *indexMapping
	if o.preprocessing {
		a, b, mapping = filterConfusingElements(a, b)
		if len(a) > 0 || len(b) > 0 {
			// Re-create context with filtered sequences
			ctx = newDiffContext(a, b, o)
		}
	}

	// Run the core algorithm
	if len(a) > 0 || len(b) > 0 {
		ctx.compareSeq(0, len(a), 0, len(b), o.forceMinimal)
	}

	// Build operations from change marks
	ops := ctx.buildOps()

	// Map indices back to original sequences
	if mapping != nil {
		ops = mapping.mapOps(ops)
	}

	// Postprocessing: shift boundaries for readability
	if o.postprocessing {
		ops = shiftBoundaries(ops, a, b)
	}

	return ops
}
