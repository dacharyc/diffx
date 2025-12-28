package diffx

// Preprocessing implementation based on concepts from:
// - Neil Fraser's "Diff Strategies" (https://neil.fraser.name/writing/diff/)
//   Describes filtering high-frequency elements that make poor alignment anchors.
// - imara-diff (Apache-2.0): https://github.com/pascalkuthe/imara-diff

// indexMapping tracks how filtered indices map back to original indices.
type indexMapping struct {
	aToOrig []int // filtered A index -> original A index
	bToOrig []int // filtered B index -> original B index
	origN   int   // original length of A
	origM   int   // original length of B
}

// mapOps converts operations on filtered sequences back to original indices.
func (m *indexMapping) mapOps(ops []DiffOp) []DiffOp {
	if m == nil {
		return ops
	}

	result := make([]DiffOp, 0, len(ops))
	for _, op := range ops {
		mapped := DiffOp{Type: op.Type}

		// Map A indices
		if op.AStart < len(m.aToOrig) {
			mapped.AStart = m.aToOrig[op.AStart]
		} else if len(m.aToOrig) > 0 {
			mapped.AStart = m.aToOrig[len(m.aToOrig)-1] + 1
		} else {
			mapped.AStart = 0
		}

		if op.AEnd > 0 && op.AEnd <= len(m.aToOrig) {
			mapped.AEnd = m.aToOrig[op.AEnd-1] + 1
		} else if op.AEnd == 0 {
			mapped.AEnd = mapped.AStart
		} else if len(m.aToOrig) > 0 {
			mapped.AEnd = m.aToOrig[len(m.aToOrig)-1] + 1
		} else {
			mapped.AEnd = m.origN
		}

		// Map B indices
		if op.BStart < len(m.bToOrig) {
			mapped.BStart = m.bToOrig[op.BStart]
		} else if len(m.bToOrig) > 0 {
			mapped.BStart = m.bToOrig[len(m.bToOrig)-1] + 1
		} else {
			mapped.BStart = 0
		}

		if op.BEnd > 0 && op.BEnd <= len(m.bToOrig) {
			mapped.BEnd = m.bToOrig[op.BEnd-1] + 1
		} else if op.BEnd == 0 {
			mapped.BEnd = mapped.BStart
		} else if len(m.bToOrig) > 0 {
			mapped.BEnd = m.bToOrig[len(m.bToOrig)-1] + 1
		} else {
			mapped.BEnd = m.origM
		}

		result = append(result, mapped)
	}

	return result
}

// elementClass indicates how an element should be treated during filtering.
type elementClass int

const (
	// keep: useful as anchor (reasonable frequency in both sequences)
	keep elementClass = iota
	// discard: definitely changed (no matches in other sequence)
	discard
	// provisional: high frequency, poor anchor but keep at boundaries
	provisional
)

// filterConfusingElements removes high-frequency elements that cause spurious matches.
// It returns filtered sequences and a mapping to convert indices back.
//
// The algorithm:
// 1. Count element frequencies in both sequences
// 2. Classify elements as keep/discard/provisional
// 3. Filter out provisional elements when surrounded by discards
// 4. Return filtered sequences with index mapping
func filterConfusingElements(a, b []Element) ([]Element, []Element, *indexMapping) {
	if len(a) == 0 || len(b) == 0 {
		return a, b, nil
	}

	// Build frequency maps using element hashes
	aFreq := make(map[uint64]int)
	bFreq := make(map[uint64]int)

	for _, e := range a {
		aFreq[e.Hash()]++
	}
	for _, e := range b {
		bFreq[e.Hash()]++
	}

	// Calculate threshold for "too common"
	// Elements appearing more than this are poor anchors
	threshold := 5 + (len(a)+len(b))/64
	if threshold < 8 {
		threshold = 8
	}

	// Classify elements in A
	aClass := make([]elementClass, len(a))
	for i, e := range a {
		h := e.Hash()
		inB := bFreq[h] > 0
		freq := aFreq[h] + bFreq[h]

		if !inB {
			aClass[i] = discard
		} else if freq > threshold {
			aClass[i] = provisional
		} else {
			aClass[i] = keep
		}
	}

	// Classify elements in B
	bClass := make([]elementClass, len(b))
	for i, e := range b {
		h := e.Hash()
		inA := aFreq[h] > 0
		freq := aFreq[h] + bFreq[h]

		if !inA {
			bClass[i] = discard
		} else if freq > threshold {
			bClass[i] = provisional
		} else {
			bClass[i] = keep
		}
	}

	// Check if filtering would help
	// If most elements would be kept, skip filtering
	keepCount := 0
	for _, c := range aClass {
		if c == keep {
			keepCount++
		}
	}
	for _, c := range bClass {
		if c == keep {
			keepCount++
		}
	}
	if keepCount > (len(a)+len(b))*3/4 {
		return a, b, nil
	}

	// Filter sequences: keep elements, discard provisionals surrounded by discards
	filteredA, aToOrig := filterSequence(a, aClass)
	filteredB, bToOrig := filterSequence(b, bClass)

	// If filtering removed everything, return original
	if len(filteredA) == 0 && len(filteredB) == 0 {
		return a, b, nil
	}

	mapping := &indexMapping{
		aToOrig: aToOrig,
		bToOrig: bToOrig,
		origN:   len(a),
		origM:   len(b),
	}

	return filteredA, filteredB, mapping
}

// filterSequence filters a sequence based on element classes.
// Provisional elements are kept only at boundaries between keep and discard regions.
func filterSequence(elems []Element, classes []elementClass) ([]Element, []int) {
	result := make([]Element, 0, len(elems))
	toOrig := make([]int, 0, len(elems))

	for i, class := range classes {
		switch class {
		case keep:
			result = append(result, elems[i])
			toOrig = append(toOrig, i)
		case provisional:
			// Keep provisional elements at boundaries
			// (when previous or next element is keep)
			atStart := i == 0
			atEnd := i == len(classes)-1
			prevKeep := !atStart && classes[i-1] == keep
			nextKeep := !atEnd && classes[i+1] == keep

			if prevKeep || nextKeep {
				result = append(result, elems[i])
				toOrig = append(toOrig, i)
			}
		// case discard: don't include
		}
	}

	return result, toOrig
}
