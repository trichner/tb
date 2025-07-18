package set

// Set is a simple set data structure that stores unique elements.
// It is implemented as a map with empty struct values to minimize memory usage.
type Set[T comparable] struct {
	elements map[T]struct{}
}

// New creates a new empty set.
func New[T comparable]() *Set[T] {
	return &Set[T]{
		elements: make(map[T]struct{}),
	}
}

// Add adds an element to the set.
// If the element already exists, it has no effect.
func (s *Set[T]) Add(item T) {
	s.elements[item] = struct{}{}
}

// ToSlice converts the set to a slice.
// The order of elements in the slice is not guaranteed.
func (s *Set[T]) ToSlice() []T {
	result := make([]T, 0, len(s.elements))
	for item := range s.elements {
		result = append(result, item)
	}
	return result
}
