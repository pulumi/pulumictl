package util

// Assert that an error will be nil.
func NoErr(err error) {
	if err != nil {
		panic("internal error: " + err.Error())
	}
}

// Ignore a value.
//
// Useful for declaring that an error is irrelevant.
func Ignore[T any](T) {}
