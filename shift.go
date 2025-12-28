package diffx

import (
	"strings"
)

// Boundary shifting preferences (higher = more preferred)
const (
	// blankLineBonus is the score bonus for keeping a blank line as a separator
	blankLineBonus = 10
	// startOfLineBonus is added when a change starts at the beginning of content
	startOfLineBonus = 3
	// endOfLineBonus is added when a change ends at the end of content
	endOfLineBonus = 3
	// punctuationBonus is added when boundary is at punctuation
	punctuationBonus = 2
)

// shiftBoundaries adjusts diff boundaries for better readability.
// When matching elements appear at boundaries, there may be multiple
// valid placements. This function shifts boundaries to prefer:
//   - Keeping blank lines as separators (not part of changes)
//   - Aligning with logical block boundaries
//   - Grouping related changes together
func shiftBoundaries(ops []DiffOp, a, b []Element) []DiffOp {
	if len(ops) == 0 {
		return ops
	}

	// First pass: shift individual operations
	result := make([]DiffOp, 0, len(ops))
	for i, op := range ops {
		if op.Type == Equal {
			result = append(result, op)
			continue
		}
		shifted := shiftOp(op, ops, i, a, b)
		result = append(result, shifted)
	}

	// Second pass: merge adjacent operations of the same type
	result = mergeAdjacentOps(result)

	// Third pass: try to improve boundaries between change regions
	result = optimizeBoundaries(result, a, b)

	return result
}

// shiftOp attempts to shift a single operation's boundaries for readability.
func shiftOp(op DiffOp, ops []DiffOp, idx int, a, b []Element) DiffOp {
	switch op.Type {
	case Delete:
		return shiftDelete(op, ops, idx, a, b)
	case Insert:
		return shiftInsert(op, ops, idx, a, b)
	default:
		return op
	}
}

// shiftDelete tries to shift deletion boundaries for better readability.
func shiftDelete(op DiffOp, ops []DiffOp, idx int, a, b []Element) DiffOp {
	if op.AEnd-op.AStart == 0 {
		return op
	}

	// Calculate how far we can shift in each direction
	maxShiftForward := 0
	maxShiftBackward := 0

	// Check forward shifting potential
	for i := 0; op.AEnd+i < len(a); i++ {
		if !a[op.AStart+i].Equal(a[op.AEnd+i]) {
			break
		}
		maxShiftForward = i + 1
	}

	// Check backward shifting potential
	for i := 0; op.AStart-i-1 >= 0; i++ {
		if !a[op.AEnd-i-1].Equal(a[op.AStart-i-1]) {
			break
		}
		maxShiftBackward = i + 1
	}

	if maxShiftForward == 0 && maxShiftBackward == 0 {
		return op
	}

	// Score each possible position
	bestShift := 0
	bestScore := scoreBoundary(op.AStart, op.AEnd, a)

	// Try forward shifts
	for shift := 1; shift <= maxShiftForward; shift++ {
		score := scoreBoundary(op.AStart+shift, op.AEnd+shift, a)
		if score > bestScore {
			bestScore = score
			bestShift = shift
		}
	}

	// Try backward shifts
	for shift := 1; shift <= maxShiftBackward; shift++ {
		score := scoreBoundary(op.AStart-shift, op.AEnd-shift, a)
		if score > bestScore {
			bestScore = score
			bestShift = -shift
		}
	}

	if bestShift == 0 {
		return op
	}

	return DiffOp{
		Type:   Delete,
		AStart: op.AStart + bestShift,
		AEnd:   op.AEnd + bestShift,
		BStart: op.BStart,
		BEnd:   op.BEnd,
	}
}

// shiftInsert tries to shift insertion boundaries for better readability.
func shiftInsert(op DiffOp, ops []DiffOp, idx int, a, b []Element) DiffOp {
	if op.BEnd-op.BStart == 0 {
		return op
	}

	// Calculate how far we can shift in each direction
	maxShiftForward := 0
	maxShiftBackward := 0

	// Check forward shifting potential
	for i := 0; op.BEnd+i < len(b); i++ {
		if !b[op.BStart+i].Equal(b[op.BEnd+i]) {
			break
		}
		maxShiftForward = i + 1
	}

	// Check backward shifting potential
	for i := 0; op.BStart-i-1 >= 0; i++ {
		if !b[op.BEnd-i-1].Equal(b[op.BStart-i-1]) {
			break
		}
		maxShiftBackward = i + 1
	}

	if maxShiftForward == 0 && maxShiftBackward == 0 {
		return op
	}

	// Score each possible position
	bestShift := 0
	bestScore := scoreBoundary(op.BStart, op.BEnd, b)

	// Try forward shifts
	for shift := 1; shift <= maxShiftForward; shift++ {
		score := scoreBoundary(op.BStart+shift, op.BEnd+shift, b)
		if score > bestScore {
			bestScore = score
			bestShift = shift
		}
	}

	// Try backward shifts
	for shift := 1; shift <= maxShiftBackward; shift++ {
		score := scoreBoundary(op.BStart-shift, op.BEnd-shift, b)
		if score > bestScore {
			bestScore = score
			bestShift = -shift
		}
	}

	if bestShift == 0 {
		return op
	}

	return DiffOp{
		Type:   Insert,
		AStart: op.AStart,
		AEnd:   op.AEnd,
		BStart: op.BStart + bestShift,
		BEnd:   op.BEnd + bestShift,
	}
}

// scoreBoundary scores a boundary position based on readability heuristics.
// Higher scores indicate better boundary positions.
func scoreBoundary(start, end int, elems []Element) int {
	score := 0

	// Bonus for blank line before the change region
	if start > 0 && isBlank(elems[start-1]) {
		score += blankLineBonus
	}

	// Bonus for blank line after the change region
	if end < len(elems) && isBlank(elems[end]) {
		score += blankLineBonus
	}

	// Bonus for starting at beginning of sequence
	if start == 0 {
		score += startOfLineBonus
	}

	// Bonus for ending at end of sequence
	if end == len(elems) {
		score += endOfLineBonus
	}

	// Check for punctuation boundaries
	if start > 0 && endsWithPunctuation(elems[start-1]) {
		score += punctuationBonus
	}
	if end < len(elems) && startsWithPunctuation(elems[end]) {
		score += punctuationBonus
	}

	return score
}

// isBlank checks if an element represents blank/whitespace content.
func isBlank(e Element) bool {
	s, ok := e.(StringElement)
	if !ok {
		return false
	}
	return strings.TrimSpace(string(s)) == ""
}

// endsWithPunctuation checks if an element ends with sentence punctuation.
func endsWithPunctuation(e Element) bool {
	s, ok := e.(StringElement)
	if !ok {
		return false
	}
	str := strings.TrimSpace(string(s))
	if len(str) == 0 {
		return false
	}
	last := str[len(str)-1]
	return last == '.' || last == '!' || last == '?' || last == ':' || last == ';'
}

// startsWithPunctuation checks if an element starts with punctuation.
func startsWithPunctuation(e Element) bool {
	s, ok := e.(StringElement)
	if !ok {
		return false
	}
	str := strings.TrimSpace(string(s))
	if len(str) == 0 {
		return false
	}
	first := str[0]
	// Common sentence starters after punctuation
	return first == '-' || first == '*' || first == '#' || first == '>'
}

// optimizeBoundaries performs a second pass to improve boundaries between
// adjacent Equal and change regions.
func optimizeBoundaries(ops []DiffOp, a, b []Element) []DiffOp {
	if len(ops) < 2 {
		return ops
	}

	result := make([]DiffOp, len(ops))
	copy(result, ops)

	// Look for patterns where we can improve boundaries
	for i := 0; i < len(result)-1; i++ {
		curr := result[i]
		next := result[i+1]

		// Pattern: Equal followed by Delete/Insert
		// Try to move blank lines from the Equal into the change if it improves things
		if curr.Type == Equal && (next.Type == Delete || next.Type == Insert) {
			result[i], result[i+1] = tryShiftEqualBoundary(curr, next, a, b)
		}

		// Pattern: Delete/Insert followed by Equal
		// Try to move blank lines from the change into the Equal
		if (curr.Type == Delete || curr.Type == Insert) && next.Type == Equal {
			result[i], result[i+1] = tryShiftChangeBoundary(curr, next, a, b)
		}
	}

	return result
}

// tryShiftEqualBoundary attempts to shift the boundary between an Equal
// region and a following change region.
func tryShiftEqualBoundary(eq, change DiffOp, a, b []Element) (DiffOp, DiffOp) {
	// Check if the last element of Equal is blank
	if eq.AEnd-eq.AStart == 0 {
		return eq, change
	}

	lastEqA := eq.AEnd - 1
	if lastEqA < 0 || lastEqA >= len(a) {
		return eq, change
	}

	// Don't move blank lines into changes - keep them as separators
	// This is actually the opposite of what we want, so just return unchanged
	return eq, change
}

// tryShiftChangeBoundary attempts to shift the boundary between a change
// region and a following Equal region.
func tryShiftChangeBoundary(change, eq DiffOp, a, b []Element) (DiffOp, DiffOp) {
	// Check if the first element of Equal is blank
	if eq.AEnd-eq.AStart == 0 {
		return change, eq
	}

	// If the change ends right before a blank line in the Equal region,
	// and we can shift to put the blank in Equal, that's better
	// (blank lines should be separators, not part of changes)

	// This is already handled by scoreBoundary, so just return unchanged
	return change, eq
}

// mergeAdjacentOps merges consecutive operations of the same type.
func mergeAdjacentOps(ops []DiffOp) []DiffOp {
	if len(ops) <= 1 {
		return ops
	}

	result := make([]DiffOp, 0, len(ops))
	current := ops[0]

	for i := 1; i < len(ops); i++ {
		op := ops[i]

		// Check if we can merge
		canMerge := current.Type == op.Type &&
			current.AEnd == op.AStart &&
			current.BEnd == op.BStart

		if canMerge {
			// Extend current operation
			current.AEnd = op.AEnd
			current.BEnd = op.BEnd
		} else {
			result = append(result, current)
			current = op
		}
	}

	result = append(result, current)
	return result
}
