package sdk

import (
	"context"

	"github.com/bavix/gripmock/v3/pkg/sdk/internal/remoteapi"
)

type remoteHistory struct {
	mock *remoteMock
}

func (r *remoteHistory) AllContext(ctx context.Context) ([]CallRecord, error) {
	return r.fetchWithClient(r.mock.apiWithContext(ctx)) //nolint:contextcheck
}

func (r *remoteHistory) CountContext(ctx context.Context) (int, error) {
	_, total, err := r.fetch(r.mock.apiWithContext(ctx), remoteapi.HistoryFilter{}) //nolint:contextcheck
	if err != nil {
		return 0, err
	}

	return total, nil
}

func (r *remoteHistory) FilterByMethodContext(ctx context.Context, svc, m string) ([]CallRecord, error) {
	calls, _, err := r.fetch( //nolint:contextcheck
		r.mock.apiWithContext(ctx),
		remoteapi.HistoryFilter{Service: svc, Method: m},
	)

	return calls, err
}

func (r *remoteHistory) fetchWithClient(client remoteapi.Client) ([]CallRecord, error) {
	calls, _, err := r.fetch(client, remoteapi.HistoryFilter{})

	return calls, err
}

func (r *remoteHistory) fetch(client remoteapi.Client, filter remoteapi.HistoryFilter) ([]CallRecord, int, error) {
	history, total, err := client.FetchHistoryFiltered(filter)
	if err != nil {
		r.mock.setOpErr(err)

		return nil, 0, err
	}

	return convertHistory(history), total, nil
}

func convertHistory(history []remoteapi.HistoryCall) []CallRecord {
	out := make([]CallRecord, len(history))
	for i, c := range history {
		out[i] = CallRecord{
			Service:         c.Service,
			Method:          c.Method,
			Session:         c.Session,
			Requests:        c.Requests,
			Responses:       c.Responses,
			ResponseHeaders: c.ResponseHeaders,
			Code:            c.Code,
			Error:           c.Error,
			ElapsedMS:       c.ElapsedMS,
			StubID:          c.StubID,
			Timestamp:       c.Timestamp,
		}
	}

	return out
}
