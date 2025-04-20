package deps

import (
	"github.com/gripmock/stuber"

	"github.com/bavix/gripmock/internal/infra/storage"
	"github.com/bavix/gripmock/internal/infra/watcher"
	"github.com/bavix/gripmock/pkg/yaml2json"
)

func (b *Builder) Budgerigar() *stuber.Budgerigar {
	b.budgerigarOnce.Do(func() {
		b.budgerigar = stuber.NewBudgerigar(b.toggles())
	})

	return b.budgerigar
}

func (b *Builder) Extender() *storage.Extender {
	b.extenderOnce.Do(func() {
		b.extender = storage.NewStub(b.Budgerigar(), yaml2json.New(), watcher.NewStubWatcher(b.config))
	})

	return b.extender
}
