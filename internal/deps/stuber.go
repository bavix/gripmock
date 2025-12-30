package deps

import (
	"context"

	internalplugins "github.com/bavix/gripmock/v3/internal/infra/plugins"
	"github.com/bavix/gripmock/v3/internal/infra/storage"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/watcher"
	"github.com/bavix/gripmock/v3/internal/infra/yaml2json"
	"github.com/bavix/gripmock/v3/pkg/plugins"
)

func (b *Builder) Budgerigar() *stuber.Budgerigar {
	b.budgerigarOnce.Do(func() {
		b.budgerigar = stuber.NewBudgerigar(b.toggles())
	})

	return b.budgerigar
}

func (b *Builder) Extender(ctx context.Context) *storage.Extender {
	b.extenderOnce.Do(func() {
		b.LoadPlugins(ctx)

		var reg plugins.Registry
		if b.pluginRegistry != nil {
			reg = b.pluginRegistry
		} else {
			reg = internalplugins.NewRegistry()
		}

		b.extender = storage.NewStub(b.Budgerigar(), yaml2json.New(reg), watcher.NewStubWatcher(b.config))
	})

	return b.extender
}
