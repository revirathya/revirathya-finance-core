package resolver

func derefOr[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}
