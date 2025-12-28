package diffx

import "hash/fnv"

// Element represents a comparable unit (line, word, token).
// Implementations must provide equality comparison and hashing.
type Element interface {
	// Equal reports whether this element is equal to another.
	Equal(other Element) bool
	// Hash returns a hash value for this element.
	// Equal elements must have equal hashes.
	Hash() uint64
}

// StringElement is the common case for line/word comparison.
type StringElement string

// Equal reports whether s equals other.
// Returns false if other is not a StringElement.
func (s StringElement) Equal(other Element) bool {
	o, ok := other.(StringElement)
	if !ok {
		return false
	}
	return s == o
}

// Hash returns a FNV-1a hash of the string.
func (s StringElement) Hash() uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// toElements converts a slice of strings to a slice of Elements.
func toElements(strs []string) []Element {
	elems := make([]Element, len(strs))
	for i, s := range strs {
		elems[i] = StringElement(s)
	}
	return elems
}
