package deps

import (
	"github.com/gripmock/stuber"

	"github.com/bavix/features"
)

func (b *Builder) toggles() features.Toggles {
	if b.config.StrictMethodTitle {
		return features.New(stuber.MethodTitle)
	}

	return features.New()
}
