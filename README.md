# diffx

A Go diffing library implementing the Myers O(ND) algorithm with heuristics for better output quality. Designed for use cases where readability of diff output matters more than minimal edit distance.

## Installation

```bash
go get github.com/dacharyc/diffx
```

## Usage

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/dacharyc/diffx"
)

func main() {
    old := []string{"The", "quick", "brown", "fox", "jumps"}
    new := []string{"A", "slow", "red", "fox", "leaps"}

    ops := diffx.Diff(old, new)

    for _, op := range ops {
        switch op.Type {
        case diffx.Equal:
            fmt.Printf("  %v\n", old[op.AStart:op.AEnd])
        case diffx.Delete:
            fmt.Printf("- %v\n", old[op.AStart:op.AEnd])
        case diffx.Insert:
            fmt.Printf("+ %v\n", new[op.BStart:op.BEnd])
        }
    }
}
```

Output:
```
- [The quick brown]
+ [A slow red]
  [fox]
- [jumps]
+ [leaps]
```

### Custom Elements

For non-string sequences, implement the `Element` interface:

```go
type Element interface {
    Equal(other Element) bool
    Hash() uint64
}
```

Then use `DiffElements`:

```go
ops := diffx.DiffElements(elementsA, elementsB)
```

### Histogram Diff

For files with many common tokens (prose, code), histogram diff often produces cleaner output:

```go
ops := diffx.DiffHistogram(a, b)
```

### Options

```go
// Force minimal edit distance (slower, but mathematically optimal)
ops := diffx.Diff(a, b, diffx.WithMinimal(true))

// Disable preprocessing (element filtering)
ops := diffx.Diff(a, b, diffx.WithPreprocessing(false))

// Disable postprocessing (boundary shifting)
ops := diffx.Diff(a, b, diffx.WithPostprocessing(false))
```

## Why diffx?

Standard diff algorithms like Myers produce the *mathematically optimal* edit sequence (minimum number of operations). However, this can result in semantically confusing output when common tokens get matched across unrelated contexts.

### The Problem

Consider diffing these two sentences:

```
Old: "The quick brown fox jumps over the lazy dog"
New: "A slow red fox leaps over the sleeping cat"
```

A naive Myers implementation might produce fragmented output like:

```
- The
+ A
  <space>
- quick
+ slow
  <space>
- brown
+ red
  <space>
  fox
  <space>
- jumps
+ leaps
  over
  <space>
  the
  <space>
- lazy
+ sleeping
  <space>
- dog
+ cat
```

The common words "the" and spaces cause spurious matches that fragment the output.

### The diffx Solution

diffx produces cleaner, grouped output:

```
- The quick brown
+ A slow red
  fox
- jumps over the lazy dog
+ leaps over the sleeping cat
```

## How It Works

diffx uses several techniques to improve output quality:

### 1. Histogram-Style Anchor Selection

Instead of matching any common element, diffx preferentially anchors on *low-frequency* elements. Common words like "the", "for", "a" are deprioritized because they appear in many unrelated contexts.

### 2. Stopword Filtering

Known stopwords (articles, prepositions, common verbs) are excluded from anchor selection entirely, preventing them from creating spurious matches.

### 3. Preprocessing

High-frequency elements that would cause noisy matches are filtered before the core algorithm runs. The results are then mapped back to original indices.

### 4. Boundary Shifting

After computing the diff, boundaries are shifted to align with logical breaks:
- Blank lines are kept as separators (not part of changes)
- Changes align with punctuation and line boundaries
- Adjacent operations are merged

### 5. Weak Anchor Elimination

Short Equal regions consisting entirely of high-frequency elements (like a lone "the" between two changes) can be converted to explicit delete/insert pairs for cleaner output.

## Comparison with go-diff

| Feature | go-diff | diffx |
|---------|---------|-------|
| Algorithm | Myers | Myers + Histogram hybrid |
| Stopword handling | None | Filtered from anchors |
| Boundary optimization | None | Shift to logical breaks |
| High-frequency filtering | None | Preprocessing step |
| Output focus | Minimal edits | Readable grouping |

### When to Use diffx

- Word-level diffs for prose or documentation
- Token-level diffs where readability matters
- Cases where common tokens cause fragmented output

### When to Use go-diff

- Character-level diffs
- Cases requiring mathematically minimal edit distance
- When you need the standard diff-match-patch feature set

## API Reference

### Types

```go
type OpType int

const (
    Equal  OpType = iota  // Elements unchanged
    Insert                 // Elements added to B
    Delete                 // Elements removed from A
)

type DiffOp struct {
    Type   OpType
    AStart int  // Start index in A (inclusive)
    AEnd   int  // End index in A (exclusive)
    BStart int  // Start index in B (inclusive)
    BEnd   int  // End index in B (exclusive)
}
```

### Functions

```go
// Diff compares two string slices
func Diff(a, b []string, opts ...Option) []DiffOp

// DiffElements compares arbitrary Element slices
func DiffElements(a, b []Element, opts ...Option) []DiffOp

// DiffHistogram uses histogram-style diff explicitly
func DiffHistogram(a, b []string, opts ...Option) []DiffOp
```

### Options

```go
func WithHeuristic(enabled bool) Option      // Speed heuristics (default: true)
func WithMinimal(minimal bool) Option        // Force minimal edit (default: false)
func WithPreprocessing(enabled bool) Option  // Element filtering (default: true)
func WithPostprocessing(enabled bool) Option // Boundary shifting (default: true)
func WithAnchorElimination(enabled bool) Option // Remove weak anchors (default: true)
```

## Performance

diffx is optimized for quality over raw speed, but remains performant:

| Input Size | Typical Time |
|------------|--------------|
| 5 elements | ~1µs |
| 100 elements | ~20µs |
| 1000 elements | ~400µs |

For large inputs with scattered changes, diffx is typically faster than character-based diff libraries because it operates at the element level.

## References

- Myers, E.W. (1986). "An O(ND) Difference Algorithm and Its Variations"
- Neil Fraser's "Diff Strategies" (https://neil.fraser.name/writing/diff/)
- Git's histogram diff algorithm
- imara-diff (https://github.com/pascalkuthe/imara-diff)

## License

MIT License - see [LICENSE](LICENSE) for details.
