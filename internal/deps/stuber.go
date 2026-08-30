package deps

import (
	"context"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/infra/storage"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
	"github.com/bavix/gripmock/v3/internal/infra/watcher"
	"github.com/bavix/gripmock/v3/internal/infra/yaml2json"
)

func (b *Builder) Budgerigar() *stuber.Budgerigar {
	b.budgerigarOnce.Do(func() {
		b.budgerigar = stuber.NewBudgerigar()
	})

	return b.budgerigar
}

func (b *Builder) Extender(ctx context.Context) *storage.Extender {
	b.extenderOnce.Do(func() {
		b.LoadPlugins(ctx)

		reg := b.pluginRegistry

		validator := newStubValidator()
		engine := template.New(context.WithoutCancel(ctx), reg)

		b.extender = storage.NewStub(
			b.Budgerigar(),
			yaml2json.New(reg),
			watcher.NewStubWatcher(b.config),
			app.StubValidator(validator, engine),
		)
	})

	return b.extender
}
