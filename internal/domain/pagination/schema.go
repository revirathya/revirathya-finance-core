package pagination

type Request struct {
	Size  int
	Token string
}

func (pr Request) NormalizedSize() (int, error) {
	return SetPageSize(pr.Size)
}
