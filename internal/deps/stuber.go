package deps

import (
	"github.com/gripmock/stuber"

	"github.com/bavix/gripmock/internal/infra/storage"
	"github.com/bavix/gripmock/pkg/yaml2json"
)

func (b *Builder) Budgerigar() *stuber.Budgerigar {
	if b.budgerigar == nil {
		b.budgerigar = stuber.NewBudgerigar(b.toggles())
	}

	return b.budgerigar
}

func (b *Builder) Extender() *storage.Extender {
	return storage.NewStub(b.Budgerigar(), yaml2json.New())
}
