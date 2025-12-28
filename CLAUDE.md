# diffx Development Guide

## Project Overview

diffx is a Go diffing library implementing the Myers O(ND) algorithm with heuristics for better output quality. It's designed for use in go-dwdiff.

**Goal**: Produce well-grouped diff output that avoids fragmenting changes around common short words.

## Quick Commands

```bash
# Run all tests
go test -v ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run comparison tool (compares against go-diff)
go run ./cmd/compare
```

## Architecture

```
diffx/
├── diffx.go        # Public API: Diff(), DiffElements(), Options
├── element.go      # Element interface, StringElement
├── context.go      # diffContext (algorithm state), partition struct
├── snake.go        # findMiddleSnake() - bidirectional Myers search
├── compare.go      # compareSeq() - divide-and-conquer
├── filter.go       # filterConfusingElements() - preprocessing
├── shift.go        # shiftBoundaries() - postprocessing
├── diffx_test.go   # Unit tests
└── cmd/compare/    # Comparison tool against other diff libraries
```

## Algorithm Pipeline

```
Input → filterConfusingElements() → compareSeq() → mapOps() → shiftBoundaries() → Output
         (preprocessing)            (core Myers)   (restore)   (postprocessing)
```

## Sources

- Myers 1986 paper: "An O(ND) Difference Algorithm and Its Variations"
- Neil Fraser's "Diff Strategies" writeup
- imara-diff (Apache-2.0): https://github.com/pascalkuthe/imara-diff

## Key Design Decisions

### Preprocessing (`filter.go`)
- Filters high-frequency elements that cause spurious matches
- Elements classified as: keep, discard, provisional
- After diff on filtered sequences, `mapOps()` expands results back to original indices
- **Critical**: Equal operations must be expanded element-by-element to interleave filtered elements

### Heuristics (`snake.go`)
- `significantMatchLen = 16` - threshold for significant diagonal runs
- Cost limit: `sqrt(n)*sqrt(m)/4`
- "Too expensive" threshold: `sqrt(n) + sqrt(m)`
- Tracks best diagonal match for fallback when cost limit exceeded

### Postprocessing (`shift.go`)
- Scores boundary positions (blank lines, punctuation, edges)
- Shifts change regions to align with logical boundaries
- Merges adjacent operations

## Testing Strategy

### Unit Tests
- Empty sequences, equal sequences, all different
- Property test: applying diff to A produces B
- Fox example: verifies "fox" preserved as anchor

### Comparison Tests (`cmd/compare`)
- Compares against `github.com/sergi/go-diff/diffmatchpatch`
- Tests: small sequences, prose, code tokens, large files
- Metrics: operation count, change regions, timing

### Key Test Cases

```go
// The motivating example - should keep "fox" as anchor
old := []string{"The", "quick", "brown", "fox", "jumps"}
new := []string{"A", "slow", "red", "fox", "leaps"}
// Expected: 2 change regions (before/after fox), not fragmented
```

## Common Issues

### Index out of bounds in shiftBoundaries
- Cause: ops have original indices but sequences were filtered
- Fix: Pass original sequences to shiftBoundaries, not filtered ones

### Missing delete/insert for filtered elements
- Cause: mapOps wasn't expanding Equal operations
- Fix: Iterate element-by-element and fill gaps with delete/insert

### Infinite recursion in findMiddleSnake
- Cause: Partition doesn't make progress
- Fix: Proper bounds checking, greedyFallback for edge cases

## Performance Targets

Based on comparison testing:
- Small (5 elements): ~1µs
- Medium (100 elements): ~20µs
- Large (1000 elements): ~400µs
- Should be 10x+ faster than go-diff on large inputs
- Should produce fewer, better-grouped change regions

## Future Improvements

1. Word-level diffing convenience function for dwdiff use case
2. More sophisticated boundary shifting (semantic awareness)
3. Fuzz testing for robustness
4. Real-world file comparison tests
