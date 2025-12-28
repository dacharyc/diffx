// Comparison tool for validating diffx output quality against other diff implementations
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/dacharyc/diffx"
	godiff "github.com/sergi/go-diff/diffmatchpatch"
)

func main() {
	// Test cases that expose fragmentation issues
	testCases := []struct {
		name string
		a, b []string
	}{
		{
			name: "Fox example (common anchor word)",
			a:    []string{"The", "quick", "brown", "fox", "jumps"},
			b:    []string{"A", "slow", "red", "fox", "leaps"},
		},
		{
			name: "Prose with common words",
			a:    strings.Split("The quick brown fox jumps over the lazy dog in the park", " "),
			b:    strings.Split("A slow red fox leaps over the sleeping cat in the garden", " "),
		},
		{
			name: "Code-like tokens",
			a:    strings.Split("func main ( ) { fmt . Println ( hello ) }", " "),
			b:    strings.Split("func main ( ) { log . Printf ( world ) }", " "),
		},
	}

	// Add a large test case
	largeA := generateLargeText(500, 0)
	largeB := generateLargeText(500, 42) // Same structure, different seed for changes
	testCases = append(testCases, struct {
		name string
		a, b []string
	}{
		name: "Large file (500 lines, scattered changes)",
		a:    largeA,
		b:    largeB,
	})

	for _, tc := range testCases {
		fmt.Printf("\n=== %s ===\n", tc.name)
		fmt.Printf("A: %d elements, B: %d elements\n", len(tc.a), len(tc.b))

		// Test diffx
		start := time.Now()
		diffxOps := diffx.Diff(tc.a, tc.b)
		diffxTime := time.Since(start)

		// Test go-diff (operates on strings, so join/split)
		dmp := godiff.New()
		start = time.Now()
		aText := strings.Join(tc.a, "\n")
		bText := strings.Join(tc.b, "\n")
		goDiffs := dmp.DiffMain(aText, bText, true)
		goDiffTime := time.Since(start)

		// Analyze diffx results
		diffxStats := analyzeDiffx(diffxOps)
		goDiffStats := analyzeGoDiff(goDiffs)

		fmt.Printf("\ndiffx:   %v\n", diffxTime)
		fmt.Printf("  Operations: %d (Equal: %d, Delete: %d, Insert: %d)\n",
			diffxStats.total, diffxStats.equal, diffxStats.delete, diffxStats.insert)
		fmt.Printf("  Change regions: %d\n", diffxStats.changeRegions)

		fmt.Printf("\ngo-diff: %v\n", goDiffTime)
		fmt.Printf("  Operations: %d (Equal: %d, Delete: %d, Insert: %d)\n",
			goDiffStats.total, goDiffStats.equal, goDiffStats.delete, goDiffStats.insert)
		fmt.Printf("  Change regions: %d\n", goDiffStats.changeRegions)

		// Show detailed output for small cases
		if len(tc.a) <= 20 {
			fmt.Println("\ndiffx output:")
			for _, op := range diffxOps {
				switch op.Type {
				case diffx.Equal:
					fmt.Printf("  = %v\n", tc.a[op.AStart:op.AEnd])
				case diffx.Delete:
					fmt.Printf("  - %v\n", tc.a[op.AStart:op.AEnd])
				case diffx.Insert:
					fmt.Printf("  + %v\n", tc.b[op.BStart:op.BEnd])
				}
			}
		}
	}
}

type diffStats struct {
	total, equal, delete, insert int
	changeRegions                int
}

func analyzeDiffx(ops []diffx.DiffOp) diffStats {
	var s diffStats
	s.total = len(ops)
	inChange := false
	for _, op := range ops {
		switch op.Type {
		case diffx.Equal:
			s.equal++
			inChange = false
		case diffx.Delete:
			s.delete++
			if !inChange {
				s.changeRegions++
				inChange = true
			}
		case diffx.Insert:
			s.insert++
			if !inChange {
				s.changeRegions++
				inChange = true
			}
		}
	}
	return s
}

func analyzeGoDiff(diffs []godiff.Diff) diffStats {
	var s diffStats
	s.total = len(diffs)
	inChange := false
	for _, d := range diffs {
		switch d.Type {
		case godiff.DiffEqual:
			s.equal++
			inChange = false
		case godiff.DiffDelete:
			s.delete++
			if !inChange {
				s.changeRegions++
				inChange = true
			}
		case godiff.DiffInsert:
			s.insert++
			if !inChange {
				s.changeRegions++
				inChange = true
			}
		}
	}
	return s
}

func generateLargeText(lines int, seed int) []string {
	words := []string{"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
		"func", "main", "return", "if", "else", "for", "range", "var", "const",
		"import", "package", "type", "struct", "interface", "map", "slice"}

	result := make([]string, lines)
	for i := 0; i < lines; i++ {
		// Generate a line with some words
		lineWords := make([]string, 5+i%3)
		for j := range lineWords {
			idx := (i*7 + j*13 + seed) % len(words)
			lineWords[j] = words[idx]
		}
		result[i] = strings.Join(lineWords, " ")
	}

	// Introduce some changes based on seed
	for i := seed % 10; i < lines; i += 10 + seed%5 {
		result[i] = "CHANGED LINE " + fmt.Sprint(i)
	}

	return result
}
