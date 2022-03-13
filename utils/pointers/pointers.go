package pointers

func ToPtr[T any](source T) *T {
	return &source
}

func FromPtr[T any](source *T) T {
	var dest T
	if source != nil {
		dest = *source
	}
	return dest
}
