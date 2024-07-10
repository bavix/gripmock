package deps

import "github.com/gripmock/environment"

func (b *Builder) Config() environment.Config {
	return b.config
}
