package util

func NoErr(err error) {
	if err != nil {
		panic("internal error: " + err.Error())
	}
}
