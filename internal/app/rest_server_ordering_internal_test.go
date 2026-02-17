package app

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestCollectAllServices_ReturnsSortedServicesAndMethods(t *testing.T) {
	t.Parallel()

	server, err := NewRestServer(t.Context(), stuber.NewBudgerigar(features.New()), &mockExtender{}, nil, nil, nil)
	require.NoError(t, err)

	services := server.collectAllServices()
	require.NotEmpty(t, services)

	serviceIDs := make([]string, 0, len(services))
	for _, service := range services {
		serviceIDs = append(serviceIDs, service.Id)

		methodIDs := make([]string, 0, len(service.Methods))
		for _, method := range service.Methods {
			methodIDs = append(methodIDs, method.Id)
		}

		require.True(t, sort.StringsAreSorted(methodIDs))
	}

	require.True(t, sort.StringsAreSorted(serviceIDs))
}
