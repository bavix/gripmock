package features

import "net/http"

type (
	Feature      string
	FeatureSlice map[Feature]bool
)

const (
	RequestInternal Feature = "X-GripMock-RequestInternal"
)

func (f FeatureSlice) Has(feature Feature) bool {
	_, ok := f[feature]

	return ok
}

func New(r *http.Request) FeatureSlice {
	return FeatureSlice{
		RequestInternal: len(r.Header.Values(string(RequestInternal))) > 0,
	}
}
