// Package util contains internal utilities.
package util

// PtrTo returns a pointer to the given value.
func PtrTo[T any](v T) *T {
	return &v
}

// ValueOrDefault returns the value from the given pointer, or the zero value of the given type.
func ValueOrDefault[T any](p *T) (v T) { //nolint:ireturn // false positive
	if p == nil {
		return
	}

	return *p
}
