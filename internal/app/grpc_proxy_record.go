package app

import (
	"google.golang.org/grpc/status"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func (m *grpcMocker) recordCapturedUnaryStub(request map[string]any, response map[string]any, callErr error, sessionID string) {
	stub := &stuber.Stub{
		Service: m.fullServiceName,
		Method:  m.methodName,
		Session: sessionID,
		Input:   stuber.InputData{Equals: request},
		Output:  stuber.Output{Data: response},
	}

	if callErr != nil {
		st := status.Convert(callErr)
		code := st.Code()
		stub.Output.Code = &code
		stub.Output.Error = st.Message()
		stub.Output.Data = nil
	}

	m.budgerigar.PutMany(stub)
}

func (m *grpcMocker) recordCapturedServerStreamStub(request map[string]any, responses []map[string]any, sessionID string) {
	streamOutput := make([]any, 0, len(responses))
	for _, response := range responses {
		streamOutput = append(streamOutput, response)
	}

	m.budgerigar.PutMany(&stuber.Stub{
		Service: m.fullServiceName,
		Method:  m.methodName,
		Session: sessionID,
		Input:   stuber.InputData{Equals: request},
		Output:  stuber.Output{Stream: streamOutput},
	})
}

func (m *grpcMocker) recordCapturedClientStreamStub(requests []map[string]any, response map[string]any, callErr error, sessionID string) {
	inputs := make([]stuber.InputData, 0, len(requests))
	for _, request := range requests {
		inputs = append(inputs, stuber.InputData{Equals: request})
	}

	stub := &stuber.Stub{
		Service: m.fullServiceName,
		Method:  m.methodName,
		Session: sessionID,
		Inputs:  inputs,
		Output:  stuber.Output{Data: response},
	}

	if callErr != nil {
		st := status.Convert(callErr)
		code := st.Code()
		stub.Output.Code = &code
		stub.Output.Error = st.Message()
		stub.Output.Data = nil
	}

	m.budgerigar.PutMany(stub)
}

func (m *grpcMocker) recordCapturedBidiStub(requests []map[string]any, responses []map[string]any, sessionID string) {
	inputs := make([]stuber.InputData, 0, len(requests))
	for _, request := range requests {
		inputs = append(inputs, stuber.InputData{Equals: request})
	}

	streamOutput := make([]any, 0, len(responses))
	for _, response := range responses {
		streamOutput = append(streamOutput, response)
	}

	m.budgerigar.PutMany(&stuber.Stub{
		Service: m.fullServiceName,
		Method:  m.methodName,
		Session: sessionID,
		Inputs:  inputs,
		Output:  stuber.Output{Stream: streamOutput},
	})
}
