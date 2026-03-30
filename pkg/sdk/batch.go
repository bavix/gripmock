package sdk

import (
	"fmt"
	"strings"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type stubCommitter interface {
	commitStubs(stubs []*stuber.Stub) error
}

// StubBatch accumulates stubs and commits them in one step.
//
// In remote mode, Commit performs a single batch REST call to reduce request count.
// In embedded mode, Commit applies stubs in-process.
type StubBatch struct {
	mock  Mock
	stubs []*stuber.Stub
}

func NewBatch(mock Mock) *StubBatch {
	return &StubBatch{mock: mock}
}

func (b *StubBatch) Stub(service, method string) StubBuilder {
	if strings.TrimSpace(service) == "" || strings.TrimSpace(method) == "" {
		panic("sdk.StubBatch.Stub: service and method must be non-empty")
	}

	return &stubBuilderCore{
		service: service,
		method:  method,
		onCommit: func(stub *stuber.Stub) {
			b.stubs = append(b.stubs, stub)
		},
	}
}

func (b *StubBatch) Commit() error {
	if len(b.stubs) == 0 {
		return nil
	}

	committer, ok := b.mock.(stubCommitter)
	if !ok {
		return fmt.Errorf("sdk: mock does not support batch commit")
	}

	if err := committer.commitStubs(b.stubs); err != nil {
		return err
	}

	b.stubs = nil

	return nil
}

func (b *StubBatch) MustCommit() {
	panicIfErr(b.Commit())
}
