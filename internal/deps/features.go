package deps

import (
	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func (b *Builder) toggles() features.Toggles {
	if b.config.StrictMethodTitle {
		return features.New(stuber.MethodTitle)
	}

	return features.New()
}
