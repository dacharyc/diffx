package diffx_test

import (
	"fmt"
	"strings"

	"github.com/dacharyc/diffx"
)

func Example() {
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
	// Output:
	// - [The quick brown]
	// + [A slow red]
	//   [fox]
	// - [jumps]
	// + [leaps]
}

func ExampleDiff() {
	old := []string{"hello", "world"}
	new := []string{"hello", "there", "world"}

	ops := diffx.Diff(old, new)

	for _, op := range ops {
		switch op.Type {
		case diffx.Equal:
			fmt.Printf("KEEP:   %v\n", old[op.AStart:op.AEnd])
		case diffx.Delete:
			fmt.Printf("DELETE: %v\n", old[op.AStart:op.AEnd])
		case diffx.Insert:
			fmt.Printf("INSERT: %v\n", new[op.BStart:op.BEnd])
		}
	}
	// Output:
	// KEEP:   [hello]
	// INSERT: [there]
	// KEEP:   [world]
}

func ExampleDiff_prose() {
	// diffx groups changes coherently, avoiding fragmentation around common words
	old := strings.Split("The quick brown fox jumps over the lazy dog", " ")
	new := strings.Split("A slow red fox leaps over the sleeping cat", " ")

	ops := diffx.Diff(old, new)

	// Count change regions (consecutive delete/insert operations)
	changeRegions := 0
	inChange := false
	for _, op := range ops {
		if op.Type == diffx.Equal {
			inChange = false
		} else if !inChange {
			changeRegions++
			inChange = true
		}
	}

	fmt.Printf("Change regions: %d\n", changeRegions)
	// Output:
	// Change regions: 3
}

func ExampleDiff_withOptions() {
	old := []string{"a", "b", "c"}
	new := []string{"a", "x", "c"}

	// Force minimal edit distance (slower but mathematically optimal)
	ops := diffx.Diff(old, new, diffx.WithMinimal(true))

	for _, op := range ops {
		fmt.Printf("%s: A[%d:%d] B[%d:%d]\n",
			op.Type, op.AStart, op.AEnd, op.BStart, op.BEnd)
	}
	// Output:
	// Equal: A[0:1] B[0:1]
	// Delete: A[1:2] B[1:1]
	// Insert: A[2:2] B[1:2]
	// Equal: A[2:3] B[2:3]
}

func ExampleDiffHistogram() {
	// Histogram diff is especially good for files with many common tokens
	old := []string{"the", "quick", "fox", "the", "end"}
	new := []string{"the", "slow", "fox", "the", "end"}

	ops := diffx.DiffHistogram(old, new)

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
	// Output:
	//   [the]
	// - [quick]
	// + [slow]
	//   [fox the end]
}

// CustomElement demonstrates implementing the Element interface
// for custom types.
type CustomElement struct {
	ID   int
	Name string
}

func (e CustomElement) Equal(other diffx.Element) bool {
	o, ok := other.(CustomElement)
	if !ok {
		return false
	}
	return e.ID == o.ID // Compare by ID only
}

func (e CustomElement) Hash() uint64 {
	return uint64(e.ID)
}

func ExampleDiffElements() {
	old := []diffx.Element{
		CustomElement{1, "Alice"},
		CustomElement{2, "Bob"},
		CustomElement{3, "Charlie"},
	}
	new := []diffx.Element{
		CustomElement{1, "Alice Smith"}, // Same ID, different name - considered equal
		CustomElement{4, "David"},       // New element
		CustomElement{3, "Charlie"},
	}

	ops := diffx.DiffElements(old, new)

	for _, op := range ops {
		switch op.Type {
		case diffx.Equal:
			fmt.Printf("KEEP:   IDs %v\n", getIDs(old[op.AStart:op.AEnd]))
		case diffx.Delete:
			fmt.Printf("DELETE: IDs %v\n", getIDs(old[op.AStart:op.AEnd]))
		case diffx.Insert:
			fmt.Printf("INSERT: IDs %v\n", getIDs(new[op.BStart:op.BEnd]))
		}
	}
	// Output:
	// KEEP:   IDs [1]
	// DELETE: IDs [2]
	// INSERT: IDs [4]
	// KEEP:   IDs [3]
}

func getIDs(elems []diffx.Element) []int {
	ids := make([]int, len(elems))
	for i, e := range elems {
		ids[i] = e.(CustomElement).ID
	}
	return ids
}

func ExampleOpType_String() {
	ops := []diffx.OpType{diffx.Equal, diffx.Insert, diffx.Delete}
	for _, op := range ops {
		fmt.Println(op.String())
	}
	// Output:
	// Equal
	// Insert
	// Delete
}
