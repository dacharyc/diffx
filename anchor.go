package diffx

// Anchor quality analysis and weak anchor elimination.
//
// The Myers diff algorithm finds the mathematically optimal edit sequence,
// but this can produce semantically incoherent output when common tokens
// (like "the", "for", "-") get matched across completely different contexts.
//
// This post-processor identifies "weak anchors" - Equal regions that are:
// 1. Short (few elements)
// 2. Contain high-frequency elements
// 3. Surrounded by changes on both sides
//
// These weak anchors are converted to Delete+Insert pairs, which produces
// cleaner, more readable output even if technically less "optimal."

// anchorOptions configures weak anchor elimination.
type anchorOptions struct {
	// maxWeakAnchorLen is the maximum length of an Equal region to consider
	// as a potential weak anchor. Longer runs are assumed to be meaningful.
	maxWeakAnchorLen int

	// frequencyThreshold is the minimum frequency (in both sequences combined)
	// for an element to be considered "high-frequency" and thus a weak anchor.
	frequencyThreshold int

	// contextWindow is how many elements before/after to check for context match.
	contextWindow int
}

func defaultAnchorOptions() *anchorOptions {
	return &anchorOptions{
		maxWeakAnchorLen:   3,  // Single elements or very short runs
		frequencyThreshold: 5,  // Elements appearing 5+ times total
		contextWindow:      2,  // Check 2 elements on each side
	}
}

// eliminateWeakAnchors was previously used to convert stopwords at Equal boundaries
// to Delete+Insert pairs. However, this created confusing self-replacement patterns
// like [-for-] {+for+} where the same word appeared both deleted and inserted.
//
// The histogram algorithm now handles stopword filtering during anchor selection,
// so this post-processing is no longer needed. We keep the function for API
// compatibility but it now just merges adjacent operations.
func eliminateWeakAnchors(ops []DiffOp, a, b []Element) []DiffOp {
	if len(ops) < 2 {
		return ops
	}

	// No longer trim stopwords - this was creating self-replacement patterns
	// The histogram algorithm already filters stopwords from being anchors

	return mergeAdjacentOps(ops)
}

// isWeakAnchor checks if an Equal region consists of high-frequency elements.
func isWeakAnchor(op DiffOp, a, b []Element, freq map[uint64]int, opts *anchorOptions) bool {
	// All elements in the Equal region must be high-frequency for it to be weak
	for i := op.AStart; i < op.AEnd; i++ {
		if i >= len(a) {
			continue
		}
		h := a[i].Hash()
		if freq[h] < opts.frequencyThreshold {
			return false // At least one low-frequency element - not weak
		}
	}
	return true
}

// hasMatchingContext checks if the surrounding context on both sides matches.
// If the elements before/after the Equal region are similar in both sequences,
// the Equal is more likely to be a meaningful anchor.
func hasMatchingContext(op DiffOp, ops []DiffOp, idx int, a, b []Element, window int) bool {
	// Get the positions in both sequences
	aPos := op.AStart
	bPos := op.BStart

	// Check context before
	matchesBefore := 0
	for i := 1; i <= window; i++ {
		aIdx := aPos - i
		bIdx := bPos - i
		if aIdx >= 0 && bIdx >= 0 && aIdx < len(a) && bIdx < len(b) {
			if a[aIdx].Equal(b[bIdx]) {
				matchesBefore++
			}
		}
	}

	// Check context after
	aEndPos := op.AEnd
	bEndPos := op.BEnd
	matchesAfter := 0
	for i := 0; i < window; i++ {
		aIdx := aEndPos + i
		bIdx := bEndPos + i
		if aIdx < len(a) && bIdx < len(b) {
			if a[aIdx].Equal(b[bIdx]) {
				matchesAfter++
			}
		}
	}

	// If we have context matches on both sides, it's probably a real anchor
	return matchesBefore > 0 && matchesAfter > 0
}

// WithAnchorElimination enables or disables weak anchor elimination.
// Default: true.
func WithAnchorElimination(enabled bool) Option {
	return func(o *options) {
		o.anchorElimination = enabled
	}
}

// trimStopwordBoundaries removes stopwords from the edges of Equal regions
// when those edges are adjacent to change operations.
//
// For example, if we have:
//   Insert["Fenestra supports a TOML configuration file following"]
//   Equal["the"]
//   Delete["background."]
//
// The "the" is a stopword at both boundaries of an Equal region adjacent to changes.
// This function converts it to Delete["the"] + Insert["the"].
//
// But if we have:
//   Equal["following the background"]
//   Delete["process"]
//
// Only "background" (if it were a stopword) at the trailing edge adjacent to Delete
// would be trimmed.
func trimStopwordBoundaries(ops []DiffOp, a, b []Element) []DiffOp {
	if len(ops) < 2 {
		return ops
	}

	result := make([]DiffOp, 0, len(ops))

	for i, op := range ops {
		if op.Type != Equal {
			result = append(result, op)
			continue
		}

		// Check if this Equal is SANDWICHED between changes (not just adjacent)
		// We only want to trim stopwords from boundaries when they're truly
		// bridging between unrelated content on both sides.
		prevIsChange := i > 0 && ops[i-1].Type != Equal
		nextIsChange := i < len(ops)-1 && ops[i+1].Type != Equal

		// Only process if sandwiched between changes on BOTH sides
		if !prevIsChange || !nextIsChange {
			// Not sandwiched - keep as is
			result = append(result, op)
			continue
		}

		// Trim stopwords from the start if previous is a change
		trimStart := 0
		if prevIsChange {
			for j := op.AStart; j < op.AEnd; j++ {
				if j < len(a) && isStopword(a[j]) {
					trimStart++
				} else {
					break
				}
			}
		}

		// Trim stopwords from the end if next is a change
		trimEnd := 0
		if nextIsChange {
			for j := op.AEnd - 1; j >= op.AStart+trimStart; j-- {
				if j < len(a) && isStopword(a[j]) {
					trimEnd++
				} else {
					break
				}
			}
		}

		// If we're trimming everything, convert entire Equal to Delete+Insert
		equalLen := op.AEnd - op.AStart
		if trimStart+trimEnd >= equalLen {
			result = append(result, DiffOp{
				Type:   Delete,
				AStart: op.AStart,
				AEnd:   op.AEnd,
				BStart: op.BStart,
				BEnd:   op.BStart,
			})
			result = append(result, DiffOp{
				Type:   Insert,
				AStart: op.AEnd,
				AEnd:   op.AEnd,
				BStart: op.BStart,
				BEnd:   op.BEnd,
			})
			continue
		}

		// Add trimmed start as Delete+Insert
		if trimStart > 0 {
			result = append(result, DiffOp{
				Type:   Delete,
				AStart: op.AStart,
				AEnd:   op.AStart + trimStart,
				BStart: op.BStart,
				BEnd:   op.BStart,
			})
			result = append(result, DiffOp{
				Type:   Insert,
				AStart: op.AStart + trimStart,
				AEnd:   op.AStart + trimStart,
				BStart: op.BStart,
				BEnd:   op.BStart + trimStart,
			})
		}

		// Add remaining Equal portion
		newAStart := op.AStart + trimStart
		newAEnd := op.AEnd - trimEnd
		newBStart := op.BStart + trimStart
		newBEnd := op.BEnd - trimEnd

		if newAEnd > newAStart {
			result = append(result, DiffOp{
				Type:   Equal,
				AStart: newAStart,
				AEnd:   newAEnd,
				BStart: newBStart,
				BEnd:   newBEnd,
			})
		}

		// Add trimmed end as Delete+Insert
		if trimEnd > 0 {
			result = append(result, DiffOp{
				Type:   Delete,
				AStart: op.AEnd - trimEnd,
				AEnd:   op.AEnd,
				BStart: op.BEnd - trimEnd,
				BEnd:   op.BEnd - trimEnd,
			})
			result = append(result, DiffOp{
				Type:   Insert,
				AStart: op.AEnd,
				AEnd:   op.AEnd,
				BStart: op.BEnd - trimEnd,
				BEnd:   op.BEnd,
			})
		}
	}

	return result
}
