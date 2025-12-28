package diffx

// Heuristic thresholds
//
// These values are independently derived based on the concepts described in:
// - Neil Fraser's "Diff Strategies" (https://neil.fraser.name/writing/diff/)
// - imara-diff (Apache-2.0): https://github.com/pascalkuthe/imara-diff
//
// The core Myers algorithm is from:
// - Myers 1986: "An O(ND) Difference Algorithm and Its Variations"
const (
	// significantMatchLen is the minimum length of a diagonal run (matching
	// elements) that indicates significant alignment progress. When we find
	// a match sequence this long, it's likely a good anchor point.
	// This value was chosen empirically - long enough to be meaningful,
	// short enough to trigger on real-world text.
	significantMatchLen = 16

	// tooExpensiveThreshold is not currently used but reserved for future
	// tuning of the cost-based early termination.
	tooExpensiveThreshold = 4
)

// snakeInfo tracks information about a diagonal run of matches found during
// the search. In Myers terminology, a "snake" is a sequence of diagonal moves
// (matching elements) in the edit graph.
type snakeInfo struct {
	x, y    int  // endpoint of the diagonal run (in local coordinates)
	len     int  // length of the match sequence
	forward bool // true if found in forward search, false if backward
}

// findMiddleSnake implements bidirectional search from Myers paper Section 4b.
// It finds the "middle snake" - the optimal split point for divide-and-conquer.
//
// Algorithm source: Myers 1986, "An O(ND) Difference Algorithm and Its Variations"
// http://www.xmailserver.org/diff2.pdf
//
// The implementation includes performance heuristics inspired by concepts from:
// - Neil Fraser's "Diff Strategies" writeup
// - imara-diff (Rust, Apache-2.0 license)
//
// Heuristics applied:
//  1. Significant match detection: Long diagonal runs indicate good alignment
//  2. Cost limit: Early termination when edit distance exceeds threshold
//  3. Expense threshold: Aggressive cutoff for pathological cases
//
// Parameters:
//   - xoff, xlim: bounds in xvec [xoff, xlim)
//   - yoff, ylim: bounds in yvec [yoff, ylim)
//   - findMinimal: if true, find the truly minimal edit path
//
// Returns a partition with the midpoint coordinates and whether each half
// needs minimal search.
func (ctx *diffContext) findMiddleSnake(xoff, xlim, yoff, ylim int, findMinimal bool) partition {
	n := xlim - xoff
	m := ylim - yoff

	// Special case: one side is empty
	if n == 0 {
		return partition{xmid: xoff, ymid: ylim, loMinimal: true, hiMinimal: true}
	}
	if m == 0 {
		return partition{xmid: xlim, ymid: yoff, loMinimal: true, hiMinimal: true}
	}

	// Delta is the difference in sequence lengths
	delta := n - m
	deltaOdd := delta&1 != 0

	// Offset for diagonal indexing (diagonals range from -m to n)
	offset := m + 1

	// Use pre-allocated diagonal arrays from context
	fdiag := ctx.fdiag
	bdiag := ctx.bdiag

	// Initialize: forward search starts at (0,0), backward at (n,m)
	fdiag[offset+1] = 0
	bdiag[offset+delta-1] = n

	// Maximum edit distance we might need to explore
	// In bidirectional search, each side explores half
	maxD := (n + m + 1) / 2

	// Apply cost limit heuristic
	costLimit := maxD
	if ctx.costLimit > 0 && !findMinimal {
		costLimit = ctx.costLimit
		if costLimit > maxD {
			costLimit = maxD
		}
	}

	// Track the best snake found (for heuristic fallback)
	var bestSnake snakeInfo
	bestSnakeScore := 0 // score = snake length, with bonus for being near middle

	// "Too expensive" threshold: if we exceed this without finding overlap,
	// use the best snake we've found
	tooExpensive := costLimit
	if ctx.useHeuristic && !findMinimal {
		// Calculate based on input size
		expensive := isqrt(n) + isqrt(m)
		if expensive < tooExpensive {
			tooExpensive = expensive
		}
	}

	for d := 0; d <= maxD; d++ {
		// Check if we've exceeded heuristic thresholds
		if ctx.useHeuristic && !findMinimal && d > tooExpensive && bestSnakeScore > 0 {
			return snakeToPartition(bestSnake, xoff, yoff, n, m)
		}

		// Forward search
		// Clamp k to valid range: k must satisfy 0 <= x <= n and 0 <= y <= m
		// Since y = x - k, we need k >= x - m and k <= x
		// Since x ranges 0..n, k ranges -m..n
		kMin := -d
		if kMin < -m {
			kMin = -m
		}
		kMax := d
		if kMax > n {
			kMax = n
		}
		// Adjust to maintain parity with d
		if (kMin+d)%2 != 0 {
			kMin++
		}

		for k := kMin; k <= kMax; k += 2 {
			kIdx := offset + k

			// Bounds check for diagonal array access
			if kIdx-1 < 0 || kIdx+1 >= len(fdiag) {
				continue
			}

			// Determine starting x: come from k-1 (insertion) or k+1 (deletion)
			var x int
			if k == -d || (k != d && fdiag[kIdx-1] < fdiag[kIdx+1]) {
				x = fdiag[kIdx+1] // From k+1, moving down (deletion from A)
			} else {
				x = fdiag[kIdx-1] + 1 // From k-1, moving right (insertion to B)
			}
			y := x - k

			// Bounds check for edit graph
			if y < 0 || y > m || x < 0 || x > n {
				fdiag[kIdx] = x // Still update to avoid stale values
				continue
			}

			// Record start of potential snake
			snakeStartX := x

			// Follow diagonal (matching elements)
			for x < n && y < m && ctx.equal(xoff+x, yoff+y) {
				x++
				y++
			}

			snakeLen := x - snakeStartX
			fdiag[kIdx] = x

			// Track significant matches for heuristic fallback
			if ctx.useHeuristic && snakeLen >= significantMatchLen {
				// Score: snake length + bonus for being near the middle
				midDist := abs((x+y)/2 - (n+m)/4)
				score := snakeLen*2 - midDist
				if score > bestSnakeScore {
					bestSnakeScore = score
					bestSnake = snakeInfo{x: x, y: y, len: snakeLen, forward: true}
				}
			}

			// Check for overlap with backward search
			// When delta is odd, we check on forward steps
			if deltaOdd && k >= delta-(d-1) && k <= delta+(d-1) {
				bIdx := offset + k - delta
				if bIdx >= 0 && bIdx < len(bdiag) && fdiag[kIdx] >= bdiag[bIdx] {
					// Found overlap - return the snake endpoint
					return partition{
						xmid:      xoff + x,
						ymid:      yoff + y,
						loMinimal: true,
						hiMinimal: true,
					}
				}
			}
		}

		// Backward search
		// Clamp k to valid range for backward search
		bkMin := -d
		if bkMin < -m {
			bkMin = -m
		}
		bkMax := d
		if bkMax > n {
			bkMax = n
		}
		// Adjust to maintain parity with d
		if (bkMin+d)%2 != 0 {
			bkMin++
		}

		for k := bkMin; k <= bkMax; k += 2 {
			kIdx := offset + k

			// Bounds check for diagonal array access
			if kIdx-1 < 0 || kIdx+1 >= len(bdiag) {
				continue
			}

			// Determine starting x
			var x int
			if k == d || (k != -d && bdiag[kIdx-1] < bdiag[kIdx+1]) {
				x = bdiag[kIdx-1] // From k-1
			} else {
				x = bdiag[kIdx+1] - 1 // From k+1
			}
			y := x - k - delta

			// Bounds check for edit graph
			if y < 0 || y > m || x < 0 || x > n {
				bdiag[kIdx] = x // Still update to avoid stale values
				continue
			}

			// Record start of potential snake (going backward)
			snakeStartX := x

			// Follow diagonal backward
			for x > 0 && y > 0 && ctx.equal(xoff+x-1, yoff+y-1) {
				x--
				y--
			}

			snakeLen := snakeStartX - x
			bdiag[kIdx] = x

			// Track significant matches for heuristic fallback
			if ctx.useHeuristic && snakeLen >= significantMatchLen {
				// Score: snake length + bonus for being near the middle
				midDist := abs((x+y)/2 - (n+m)/4)
				score := snakeLen*2 - midDist
				if score > bestSnakeScore {
					bestSnakeScore = score
					bestSnake = snakeInfo{x: x, y: y, len: snakeLen, forward: false}
				}
			}

			// Check for overlap with forward search
			// When delta is even, we check on backward steps
			if !deltaOdd && k+delta >= -d && k+delta <= d {
				fIdx := offset + k + delta
				if fIdx >= 0 && fIdx < len(fdiag) && fdiag[fIdx] >= bdiag[kIdx] {
					// Found overlap
					fx := fdiag[fIdx]
					fy := fx - (k + delta)
					return partition{
						xmid:      xoff + fx,
						ymid:      yoff + fy,
						loMinimal: true,
						hiMinimal: true,
					}
				}
			}
		}

		// Check cost limit (distinct from "too expensive")
		if d >= costLimit && bestSnakeScore > 0 {
			return snakeToPartition(bestSnake, xoff, yoff, n, m)
		}
	}

	// If we reach here, we've exhausted the search without finding overlap
	// This can happen with cost limits. Use the best snake if we have one.
	if bestSnakeScore > 0 {
		return snakeToPartition(bestSnake, xoff, yoff, n, m)
	}

	// Last resort: greedy fallback that guarantees progress
	return greedyFallback(ctx, xoff, xlim, yoff, ylim)
}

// snakeToPartition converts a diagonal match run into a partition for divide-and-conquer.
func snakeToPartition(snake snakeInfo, xoff, yoff, n, m int) partition {
	if snake.forward {
		// Forward snake: split at the end of the snake
		return partition{
			xmid:      xoff + snake.x,
			ymid:      yoff + snake.y,
			loMinimal: true,
			hiMinimal: false, // Upper half may not be minimal
		}
	}
	// Backward snake: split at the start of the snake
	return partition{
		xmid:      xoff + snake.x,
		ymid:      yoff + snake.y,
		loMinimal: false, // Lower half may not be minimal
		hiMinimal: true,
	}
}

// greedyFallback provides a simple split when the optimal search fails.
// It finds matches from the start, or makes one deletion to ensure progress.
func greedyFallback(ctx *diffContext, xoff, xlim, yoff, ylim int) partition {
	n := xlim - xoff
	m := ylim - yoff

	// Try to find matches from the start
	x := 0
	y := 0
	for x < n && y < m && ctx.equal(xoff+x, yoff+y) {
		x++
		y++
	}

	// If we found matches, split there
	if x > 0 {
		return partition{
			xmid:      xoff + x,
			ymid:      yoff + y,
			loMinimal: false,
			hiMinimal: false,
		}
	}

	// No matches at start - we need to make an edit to progress
	// Prefer deletion (consuming from x) over insertion
	if n > 0 {
		return partition{
			xmid:      xoff + 1,
			ymid:      yoff,
			loMinimal: false,
			hiMinimal: false,
		}
	}

	// n == 0, so all of y must be inserted (shouldn't reach here normally)
	return partition{
		xmid:      xoff,
		ymid:      yoff + 1,
		loMinimal: false,
		hiMinimal: false,
	}
}

// isqrt computes integer square root using Newton's method.
func isqrt(n int) int {
	if n <= 0 {
		return 0
	}
	if n == 1 {
		return 1
	}

	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}

// abs returns the absolute value of x.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
