package diffx

// Anchor elimination post-processing.
//
// The histogram algorithm handles stopword filtering during anchor selection,
// so aggressive post-processing is no longer needed. This file provides the
// eliminateWeakAnchors function which now simply merges adjacent operations.

// eliminateWeakAnchors merges adjacent operations of the same type.
//
// Previously this converted stopwords at Equal boundaries to Delete+Insert pairs,
// but that created confusing self-replacement patterns like [-for-] {+for+}.
// The histogram algorithm now filters stopwords from being anchors, so this
// post-processing just ensures operations are properly merged.
func eliminateWeakAnchors(ops []DiffOp, a, b []Element) []DiffOp {
	if len(ops) < 2 {
		return ops
	}

	return mergeAdjacentOps(ops)
}

// WithAnchorElimination enables or disables anchor elimination post-processing.
// Default: true.
func WithAnchorElimination(enabled bool) Option {
	return func(o *options) {
		o.anchorElimination = enabled
	}
}
