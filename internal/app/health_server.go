package app

import (
	"context"

	"github.com/bavix/gripmock/pkg/api/health"
)

type HealthcheckServer struct{}

func (HealthcheckServer) Liveness(_ context.Context) (*api.MessageOK, error) {
	return &api.MessageOK{}, nil
}

func (HealthcheckServer) Readiness(_ context.Context) (*api.MessageOK, error) {
	return &api.MessageOK{}, nil
}
