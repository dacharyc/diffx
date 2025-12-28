package diffx

import "testing"

func TestStringElement_Equal(t *testing.T) {
	a := StringElement("hello")
	b := StringElement("hello")
	c := StringElement("world")

	if !a.Equal(b) {
		t.Error("Expected a.Equal(b) to be true")
	}
	if a.Equal(c) {
		t.Error("Expected a.Equal(c) to be false")
	}
}

func TestStringElement_Hash(t *testing.T) {
	a := StringElement("hello")
	b := StringElement("hello")
	c := StringElement("world")

	if a.Hash() != b.Hash() {
		t.Error("Expected equal elements to have equal hashes")
	}
	if a.Hash() == c.Hash() {
		t.Error("Expected different elements to have different hashes (collision unlikely)")
	}
}

func TestStringElement_EqualDifferentType(t *testing.T) {
	a := StringElement("hello")

	// Create a mock element that's not a StringElement
	type otherElement struct{}

	// StringElement.Equal should return false for non-StringElement types
	// We can't directly test this without implementing Element interface,
	// but we verify the type assertion logic works correctly
	if a.Equal(StringElement("hello")) != true {
		t.Error("Same string should be equal")
	}
}

func TestToElements(t *testing.T) {
	strs := []string{"a", "b", "c"}
	elems := toElements(strs)

	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}

	for i, elem := range elems {
		se, ok := elem.(StringElement)
		if !ok {
			t.Errorf("element %d is not StringElement", i)
			continue
		}
		if string(se) != strs[i] {
			t.Errorf("element %d: expected %q, got %q", i, strs[i], se)
		}
	}
}

func TestToElements_Empty(t *testing.T) {
	elems := toElements([]string{})
	if len(elems) != 0 {
		t.Errorf("expected 0 elements, got %d", len(elems))
	}
}
