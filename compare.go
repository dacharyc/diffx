package diffx

// compareSeq is the divide-and-conquer core of the Myers diff algorithm.
// It compares xvec[xoff:xlim] with yvec[yoff:ylim] and marks changes
// in xchanges and ychanges.
//
// Parameters:
//   - xoff, xlim: bounds in xvec [xoff, xlim)
//   - yoff, ylim: bounds in yvec [yoff, ylim)
//   - findMinimal: if true, find the truly minimal edit script
func (ctx *diffContext) compareSeq(xoff, xlim, yoff, ylim int, findMinimal bool) {
	// 1. Trim matching elements from the start
	for xoff < xlim && yoff < ylim && ctx.equal(xoff, yoff) {
		xoff++
		yoff++
	}

	// 2. Trim matching elements from the end
	for xoff < xlim && yoff < ylim && ctx.equal(xlim-1, ylim-1) {
		xlim--
		ylim--
	}

	// 3. Base cases: one sequence is empty
	if xoff == xlim {
		// All remaining y elements are insertions
		ctx.markInserted(yoff, ylim)
		return
	}
	if yoff == ylim {
		// All remaining x elements are deletions
		ctx.markDeleted(xoff, xlim)
		return
	}

	// 4. Find the middle snake (optimal split point)
	part := ctx.findMiddleSnake(xoff, xlim, yoff, ylim, findMinimal)

	// 5. Recurse on both halves
	// Process smaller subproblem first for better memory behavior
	ctx.compareSeq(xoff, part.xmid, yoff, part.ymid, part.loMinimal)
	ctx.compareSeq(part.xmid, xlim, part.ymid, ylim, part.hiMinimal)
}

// buildOps converts the change marks into a sequence of DiffOp.
// It walks through both sequences and groups consecutive changes.
func (ctx *diffContext) buildOps() []DiffOp {
	var ops []DiffOp
	n := len(ctx.xvec)
	m := len(ctx.yvec)
	i, j := 0, 0

	for i < n || j < m {
		// Find equal prefix
		eqStart := i
		eqJStart := j
		for i < n && j < m && !ctx.xchanges[i] && !ctx.ychanges[j] {
			i++
			j++
		}
		if i > eqStart {
			ops = append(ops, DiffOp{
				Type:   Equal,
				AStart: eqStart,
				AEnd:   i,
				BStart: eqJStart,
				BEnd:   j,
			})
		}

		// Find deletions (changed in x)
		delStart := i
		for i < n && ctx.xchanges[i] {
			i++
		}
		if i > delStart {
			ops = append(ops, DiffOp{
				Type:   Delete,
				AStart: delStart,
				AEnd:   i,
				BStart: j,
				BEnd:   j,
			})
		}

		// Find insertions (changed in y)
		insStart := j
		for j < m && ctx.ychanges[j] {
			j++
		}
		if j > insStart {
			ops = append(ops, DiffOp{
				Type:   Insert,
				AStart: i,
				AEnd:   i,
				BStart: insStart,
				BEnd:   j,
			})
		}
	}

	return ops
}
