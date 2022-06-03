package pgparty

func UniqAdd[T comparable](sl []T, av T) []T {
	fnd := false
	for _, v := range sl {
		if av == v {
			fnd = true
			break
		}
	}
	if !fnd {
		sl = append(sl, av)
	}
	return sl
}
