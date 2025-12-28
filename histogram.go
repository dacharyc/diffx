package diffx

// Histogram-style diff algorithm.
//
// This implements an approach similar to Git's histogram diff:
// 1. Count element frequencies in both sequences
// 2. Find the lowest-frequency element that appears in both (the best anchor)
// 3. Split sequences at that anchor point
// 4. Recursively apply to both halves
// 5. Fall back to Myers when no good anchors exist
//
// This naturally avoids matching high-frequency elements like "the", "for", "-"
// because they're never chosen as anchor points.
//
// References:
// - JGit HistogramDiff (Eclipse License)
// - raygard/hdiff (0BSD License)
// - Bram Cohen's patience diff concept

// histogramOptions configures histogram diff behavior.
type histogramOptions struct {
	// maxChainLength is the maximum frequency for an element to be considered
	// as an anchor. Elements appearing more than this are ignored.
	// Git uses 64 by default, but for word-level diff we use a lower value.
	maxChainLength int

	// fallbackToMyers controls whether to use Myers when no good anchors exist.
	fallbackToMyers bool

	// filterStopwords prevents common words from being used as anchors.
	filterStopwords bool
}

func defaultHistogramOptions() *histogramOptions {
	return &histogramOptions{
		maxChainLength:  64, // Match Git's default; allow higher-frequency anchors
		fallbackToMyers: true,
		filterStopwords: true, // Filter stopwords for histogram anchors; Myers fallback finds others
	}
}

// stopwords are common words that make poor anchors even at low frequency.
// These words appear frequently in natural language but carry little semantic meaning.
// NOTE: We intentionally exclude single-character punctuation and code keywords
// because they ARE meaningful anchors in code diffs (e.g., matching "(" is important).
var stopwords = map[string]bool{
	// Articles and determiners - these truly have no semantic meaning
	"a": true, "an": true, "the": true,
	// Very common prepositions that often appear in unrelated contexts
	"in": true, "on": true, "to": true, "for": true, "of": true, "with": true,
	// Common conjunctions
	"and": true, "or": true,
	// Very common verbs that don't carry much meaning
	"is": true, "are": true, "be": true,
}

// isStopword checks if a string element is a stopword.
func isStopword(e Element) bool {
	s, ok := e.(StringElement)
	if !ok {
		return false
	}
	return stopwords[string(s)]
}

// histogramDiff performs histogram-style diff on two element sequences.
func histogramDiff(a, b []Element, opts *histogramOptions) []DiffOp {
	if opts == nil {
		opts = defaultHistogramOptions()
	}

	// Handle trivial cases
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	if len(a) == 0 {
		return []DiffOp{{Type: Insert, AStart: 0, AEnd: 0, BStart: 0, BEnd: len(b)}}
	}
	if len(b) == 0 {
		return []DiffOp{{Type: Delete, AStart: 0, AEnd: len(a), BStart: 0, BEnd: 0}}
	}

	// Trim common prefix
	prefixLen := 0
	for prefixLen < len(a) && prefixLen < len(b) && a[prefixLen].Equal(b[prefixLen]) {
		prefixLen++
	}

	// Trim common suffix
	suffixLen := 0
	for suffixLen < len(a)-prefixLen && suffixLen < len(b)-prefixLen &&
		a[len(a)-1-suffixLen].Equal(b[len(b)-1-suffixLen]) {
		suffixLen++
	}

	// If everything matches, return Equal
	if prefixLen+suffixLen >= len(a) && prefixLen+suffixLen >= len(b) {
		return []DiffOp{{Type: Equal, AStart: 0, AEnd: len(a), BStart: 0, BEnd: len(b)}}
	}

	// Work on the middle section
	aStart, aEnd := prefixLen, len(a)-suffixLen
	bStart, bEnd := prefixLen, len(b)-suffixLen

	// Build result with prefix
	var result []DiffOp
	if prefixLen > 0 {
		result = append(result, DiffOp{Type: Equal, AStart: 0, AEnd: prefixLen, BStart: 0, BEnd: prefixLen})
	}

	// Diff the middle section
	middleOps := histogramDiffRecursive(a[aStart:aEnd], b[bStart:bEnd], aStart, bStart, opts)
	result = append(result, middleOps...)

	// Add suffix
	if suffixLen > 0 {
		result = append(result, DiffOp{
			Type:   Equal,
			AStart: len(a) - suffixLen,
			AEnd:   len(a),
			BStart: len(b) - suffixLen,
			BEnd:   len(b),
		})
	}

	return mergeAdjacentOps(result)
}

// histogramDiffRecursive performs the core histogram algorithm on a section.
func histogramDiffRecursive(a, b []Element, aOffset, bOffset int, opts *histogramOptions) []DiffOp {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	if len(a) == 0 {
		return []DiffOp{{Type: Insert, AStart: aOffset, AEnd: aOffset, BStart: bOffset, BEnd: bOffset + len(b)}}
	}
	if len(b) == 0 {
		return []DiffOp{{Type: Delete, AStart: aOffset, AEnd: aOffset + len(a), BStart: bOffset, BEnd: bOffset}}
	}

	// Build frequency histogram for sequence A
	aFreq := make(map[uint64]int)
	aIndices := make(map[uint64][]int) // hash -> list of indices in A
	for i, e := range a {
		h := e.Hash()
		aFreq[h]++
		aIndices[h] = append(aIndices[h], i)
	}

	// Find the best anchor: consider both frequency AND position balance.
	// We want low-frequency tokens, but also tokens that create balanced splits.
	// Score = frequency * (1 + positionImbalance), lower is better.
	bestIdx := -1
	bestScore := float64(opts.maxChainLength+1) * 3 // Initialize to impossible value
	var bestHash uint64

	for i, e := range b {
		// Skip stopwords if filtering is enabled
		if opts.filterStopwords && isStopword(e) {
			continue
		}

		h := e.Hash()
		freq := aFreq[h]
		if freq <= 0 || freq > opts.maxChainLength {
			continue
		}

		// Find the best matching position in A for this potential anchor
		bRatio := float64(i) / float64(len(b))
		bestPosImbalance := 2.0

		for _, aIdx := range aIndices[h] {
			if !a[aIdx].Equal(e) {
				continue
			}
			aRatio := float64(aIdx) / float64(len(a))
			imbalance := aRatio - bRatio
			if imbalance < 0 {
				imbalance = -imbalance
			}
			if imbalance < bestPosImbalance {
				bestPosImbalance = imbalance
			}
		}

		if bestPosImbalance > 1.5 {
			continue // No valid match position found
		}

		// Score combines frequency and position imbalance
		// Lower frequency is better, lower imbalance is better
		score := float64(freq) * (1.0 + bestPosImbalance*2)

		if score < bestScore {
			bestScore = score
			bestIdx = i
			bestHash = h
		}
	}

	// No good anchor found - fall back to Myers to find more anchors.
	// Myers will find common subsequences that histogram missed.
	// The anchor elimination post-processing will clean up bad stopword matches.
	if bestIdx == -1 {
		if opts.fallbackToMyers {
			return myersFallback(a, b, aOffset, bOffset)
		}
		// Only use delete+insert if Myers fallback is disabled
		return []DiffOp{
			{Type: Delete, AStart: aOffset, AEnd: aOffset + len(a), BStart: bOffset, BEnd: bOffset},
			{Type: Insert, AStart: aOffset + len(a), AEnd: aOffset + len(a), BStart: bOffset, BEnd: bOffset + len(b)},
		}
	}

	// Find the best matching position in A for this anchor.
	// Instead of picking the first occurrence, pick the one that creates
	// the most balanced split (position ratio in A closest to position ratio in B).
	bRatio := float64(bestIdx) / float64(len(b))
	aMatchIdx := -1
	bestRatioDiff := 2.0 // Initialize to impossible value

	for _, idx := range aIndices[bestHash] {
		// Verify hash collision
		if !a[idx].Equal(b[bestIdx]) {
			continue
		}
		aRatio := float64(idx) / float64(len(a))
		ratioDiff := aRatio - bRatio
		if ratioDiff < 0 {
			ratioDiff = -ratioDiff
		}
		if ratioDiff < bestRatioDiff {
			bestRatioDiff = ratioDiff
			aMatchIdx = idx
		}
	}

	if aMatchIdx == -1 {
		// No valid match found - fall back
		if opts.fallbackToMyers {
			return myersFallback(a, b, aOffset, bOffset)
		}
		return []DiffOp{
			{Type: Delete, AStart: aOffset, AEnd: aOffset + len(a), BStart: bOffset, BEnd: bOffset},
			{Type: Insert, AStart: aOffset + len(a), AEnd: aOffset + len(a), BStart: bOffset, BEnd: bOffset + len(b)},
		}
	}

	// Extend the match forward and backward to find the full matching region
	matchStartA, matchStartB := aMatchIdx, bestIdx
	matchEndA, matchEndB := aMatchIdx+1, bestIdx+1

	// Extend backward
	for matchStartA > 0 && matchStartB > 0 && a[matchStartA-1].Equal(b[matchStartB-1]) {
		matchStartA--
		matchStartB--
	}

	// Extend forward
	for matchEndA < len(a) && matchEndB < len(b) && a[matchEndA].Equal(b[matchEndB]) {
		matchEndA++
		matchEndB++
	}

	// Recursively diff the sections before and after the match
	var result []DiffOp

	// Before the match
	if matchStartA > 0 || matchStartB > 0 {
		beforeOps := histogramDiffRecursive(
			a[:matchStartA], b[:matchStartB],
			aOffset, bOffset,
			opts,
		)
		result = append(result, beforeOps...)
	}

	// The matching region
	result = append(result, DiffOp{
		Type:   Equal,
		AStart: aOffset + matchStartA,
		AEnd:   aOffset + matchEndA,
		BStart: bOffset + matchStartB,
		BEnd:   bOffset + matchEndB,
	})

	// After the match
	if matchEndA < len(a) || matchEndB < len(b) {
		afterOps := histogramDiffRecursive(
			a[matchEndA:], b[matchEndB:],
			aOffset+matchEndA, bOffset+matchEndB,
			opts,
		)
		result = append(result, afterOps...)
	}

	return result
}

// myersFallback uses the standard Myers algorithm for a section.
func myersFallback(a, b []Element, aOffset, bOffset int) []DiffOp {
	// Create a temporary context for Myers diff
	o := defaultOptions()
	o.preprocessing = false  // Already preprocessed
	o.postprocessing = false // Will be done after
	o.anchorElimination = false

	ctx := newDiffContext(a, b, o)
	ctx.compareSeq(0, len(a), 0, len(b), false)
	ops := ctx.buildOps()

	// Adjust offsets
	for i := range ops {
		ops[i].AStart += aOffset
		ops[i].AEnd += aOffset
		ops[i].BStart += bOffset
		ops[i].BEnd += bOffset
	}

	return ops
}

// DiffHistogram performs histogram-style diff on string slices.
func DiffHistogram(a, b []string, opts ...Option) []DiffOp {
	return DiffElementsHistogram(toElements(a), toElements(b), opts...)
}

// DiffElementsHistogram performs histogram-style diff on Element slices.
func DiffElementsHistogram(a, b []Element, opts ...Option) []DiffOp {
	// Apply options
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	origA, origB := a, b

	histOpts := defaultHistogramOptions()

	// Run histogram diff
	ops := histogramDiff(a, b, histOpts)

	// Apply anchor elimination if enabled
	if o.anchorElimination {
		ops = eliminateWeakAnchors(ops, origA, origB)
	}

	// Apply boundary shifting if enabled
	if o.postprocessing {
		ops = shiftBoundaries(ops, origA, origB)
	}

	return ops
}
