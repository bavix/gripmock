package deps

import (
	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func (b *Builder) toggles() features.Toggles {
	// Enable method title normalization only when explicitly requested.
	if b.config.StrictMethodTitle { //nolint:staticcheck
		return features.New(stuber.MethodTitle)
	}

	return features.New()
}
