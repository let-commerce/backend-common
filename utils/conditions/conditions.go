package conditions

// IfThen evaluates a condition, if true returns the parameters otherwise nil
func IfThen[T any](condition bool, a T) T {
	var res T
	if condition {
		return a
	}
	return res
}

// IfThenElse evaluates a condition, if true returns the first parameter otherwise the second
func IfThenElse[T any](condition bool, a T, b T) T {
	if condition {
		return a
	}
	return b
}

// DefaultIfNil checks if the value is nil, if true returns the default value otherwise the original
func DefaultIfNil[T any](value *T, defaultValue T) T {
	if value != nil {
		return *value
	}
	return defaultValue
}

// FirstNonNil returns the first non nil parameter
func FirstNonNil[T any](values ...*T) T {
	var res T
	for _, value := range values {
		if value != nil {
			return *value
		}
	}
	return res
}
