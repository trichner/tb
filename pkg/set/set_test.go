package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	s := New[string]()
	assert.NotNil(t, s)
}

func TestAdd(t *testing.T) {
	s := New[string]()

	// Add elements
	s.Add("a")
	s.Add("b")

	// Add existing element (should have no effect)
	s.Add("a")

	// Verify through ToSlice
	slice := s.ToSlice()
	assert.Equal(t, 2, len(slice))
	assert.Contains(t, slice, "a")
	assert.Contains(t, slice, "b")
}

func TestToSlice(t *testing.T) {
	s := New[string]()
	s.Add("a")
	s.Add("b")
	s.Add("c")

	slice := s.ToSlice()

	assert.Equal(t, 3, len(slice))
	assert.Contains(t, slice, "a")
	assert.Contains(t, slice, "b")
	assert.Contains(t, slice, "c")
}
