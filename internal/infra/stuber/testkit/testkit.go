package testkit

import (
	"github.com/google/uuid"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func NewTestStub(service, method string, priority int) *stuber.Stub {
	return &stuber.Stub{
		ID:       uuid.New(),
		Service:  service,
		Method:   method,
		Priority: priority,
	}
}

func NewTestStubWithID(id uuid.UUID, service, method string, priority int) *stuber.Stub {
	return &stuber.Stub{
		ID:       id,
		Service:  service,
		Method:   method,
		Priority: priority,
	}
}
