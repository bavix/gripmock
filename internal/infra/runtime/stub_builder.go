package runtime

import (
	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

// StubBuilder builds StubStrict step by step.
type StubBuilder struct {
	stub domain.StubStrict
}

// NewStubBuilder creates a new stub builder.
func NewStubBuilder() *StubBuilder {
	return &StubBuilder{
		stub: domain.StubStrict{},
	}
}

// Service sets the service name.
func (sb *StubBuilder) Service(service string) *StubBuilder {
	sb.stub.Service = service

	return sb
}

// Method sets the method name.
func (sb *StubBuilder) Method(method string) *StubBuilder {
	sb.stub.Method = method

	return sb
}

// Priority sets the priority.
func (sb *StubBuilder) Priority(priority int) *StubBuilder {
	sb.stub.Priority = priority

	return sb
}

// Times sets the times.
func (sb *StubBuilder) Times(times int) *StubBuilder {
	sb.stub.Times = times

	return sb
}

// ID sets the ID.
func (sb *StubBuilder) ID(id string) *StubBuilder {
	sb.stub.ID = id

	return sb
}

// AddInput adds an input matcher.
func (sb *StubBuilder) AddInput(input domain.Matcher) *StubBuilder {
	sb.stub.Inputs = append(sb.stub.Inputs, input)

	return sb
}

// Headers sets the headers matcher.
func (sb *StubBuilder) Headers(headers domain.Matcher) *StubBuilder {
	sb.stub.Headers = &headers

	return sb
}

// AddResponseHeader adds a response header.
func (sb *StubBuilder) AddResponseHeader(key, value string) *StubBuilder {
	if sb.stub.ResponseHeaders == nil {
		sb.stub.ResponseHeaders = make(map[string]string)
	}

	sb.stub.ResponseHeaders[key] = value

	return sb
}

// AddResponseTrailer adds a response trailer.
func (sb *StubBuilder) AddResponseTrailer(key, value string) *StubBuilder {
	if sb.stub.ResponseTrailers == nil {
		sb.stub.ResponseTrailers = make(map[string]string)
	}

	sb.stub.ResponseTrailers[key] = value

	return sb
}

// AddDataOutput adds a data output.
func (sb *StubBuilder) AddDataOutput(content map[string]any, headers map[string]string) *StubBuilder {
	output := domain.OutputStrict{
		Data: &domain.DataResponse{
			Content: content,
			Headers: headers,
		},
	}
	sb.stub.Outputs = append(sb.stub.Outputs, output)

	return sb
}

// AddStreamOutput adds a stream output.
func (sb *StubBuilder) AddStreamOutput(steps []domain.StreamStepStrict) *StubBuilder {
	output := domain.OutputStrict{
		Stream: steps,
	}
	sb.stub.Outputs = append(sb.stub.Outputs, output)

	return sb
}

// AddDelayOutput adds a delay output.
func (sb *StubBuilder) AddDelayOutput(duration string) *StubBuilder {
	output := domain.OutputStrict{
		Delay: &domain.Delay{
			Duration: duration,
		},
	}
	sb.stub.Outputs = append(sb.stub.Outputs, output)

	return sb
}

// AddStatusOutput adds a status output.
func (sb *StubBuilder) AddStatusOutput(code, message string) *StubBuilder {
	output := domain.OutputStrict{
		Status: &domain.GrpcStatus{
			Code:    code,
			Message: message,
		},
	}
	sb.stub.Outputs = append(sb.stub.Outputs, output)

	return sb
}

// Build returns the built stub.
func (sb *StubBuilder) Build() domain.StubStrict {
	return sb.stub
}

// StreamStepBuilder builds StreamStepStrict.
type StreamStepBuilder struct {
	step domain.StreamStepStrict
}

// NewStreamStepBuilder creates a new stream step builder.
func NewStreamStepBuilder() *StreamStepBuilder {
	return &StreamStepBuilder{
		step: domain.StreamStepStrict{},
	}
}

// Send adds a send operation.
func (ssb *StreamStepBuilder) Send(data map[string]any, headers map[string]string) *StreamStepBuilder {
	ssb.step.Send = &domain.SendStep{
		Data:    data,
		Headers: headers,
	}

	return ssb
}

// Delay adds a delay operation.
func (ssb *StreamStepBuilder) Delay(duration string) *StreamStepBuilder {
	ssb.step.Delay = &domain.Delay{
		Duration: duration,
	}

	return ssb
}

// End adds an end operation.
func (ssb *StreamStepBuilder) End(code, message string) *StreamStepBuilder {
	ssb.step.End = &domain.EndStep{
		Code:    code,
		Message: message,
	}

	return ssb
}

// Build returns the built stream step.
func (ssb *StreamStepBuilder) Build() domain.StreamStepStrict {
	return ssb.step
}
