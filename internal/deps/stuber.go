package deps

import (
	"github.com/bavix/gripmock/v3/internal/infra/storage"
	localstuber "github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/watcher"
	"github.com/bavix/gripmock/v3/internal/infra/yaml2json"
)

func (b *Builder) Budgerigar() *localstuber.Budgerigar {
	b.budgerigarOnce.Do(func() {
		b.budgerigar = localstuber.NewBudgerigar(b.toggles())
	})

	return b.budgerigar
}

func (b *Builder) Extender() *storage.Extender {
	b.extenderOnce.Do(func() {
		b.extender = storage.NewStub(b.Budgerigar(), yaml2json.New(b.pluginRegistry), watcher.NewStubWatcher(b.config))
	})

	return b.extender
}
