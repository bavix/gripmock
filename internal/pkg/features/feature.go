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
	if val, ok := f[feature]; ok {
		return val
	}

	return false
}

func New(r *http.Request) FeatureSlice {
	return FeatureSlice{
		RequestInternal: len(r.Header.Values(string(RequestInternal))) > 0,
	}
}
