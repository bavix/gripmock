package stuber

import (
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/v3/internal/domain/types"
	"github.com/bavix/gripmock/v3/internal/infra/template"
	gptypes "github.com/bavix/gripmock/v3/internal/infra/types"
)

// Stub represents a gRPC service method and its associated data.
type Stub struct {
	ID               uuid.UUID         `json:"id"`                         // The unique identifier of the stub.
	Service          string            `json:"service"`                    // The name of the service.
	Method           string            `json:"method"`                     // The name of the method.
	Priority         int               `json:"priority"`                   // The priority score of the stub.
	Times            int               `json:"times,omitempty"`            // Maximum successful triggers for this stub (0 = unlimited).
	ResponseHeaders  map[string]string `json:"responseHeaders,omitempty"`  // Initial response metadata to send once before the first response.
	ResponseTrailers map[string]string `json:"responseTrailers,omitempty"` // Trailing metadata to send once when RPC completes.
	Headers          InputHeader       `json:"headers"`                    // The headers of the request.
	Input            InputData         `json:"input"`                      // Input data for unary requests
	Inputs           []InputData       `json:"inputs,omitempty"`           // Input data for client streaming requests
	Output           Output            `json:"output"`                     // The output data of the response.
	// V4 fields for new format support
	InputsV4     []types.Matcher  `json:"-"` // V4 input matchers (plural) - populated by custom unmarshaling
	OutputsRawV4 []map[string]any `json:"-"` // V4 output rules (plural) - populated by custom unmarshaling
	OutputsV4    []OutputV4       `json:"-"` // V4 output rules (plural) - typed version
}

// OutputV4 represents a single v4 output rule that can be one of three types.
type OutputV4 struct {
	// Data output for unary responses
	Data *types.DataRule `json:"data,omitempty"`

	// Stream output for server/bidirectional streaming
	Stream *types.StreamRule `json:"stream,omitempty"`

	// Sequence output for complex multi-step responses
	Sequence *types.SequenceRule `json:"sequence,omitempty"`

	// Raw data for backward compatibility
	Raw map[string]any `json:"-"`
}

// IsUnary returns true if this stub is for unary requests (has Input data).
// For v4 stubs, this is determined by the method registry.
// For legacy stubs, this is determined by having Input data (no Inputs).
func (s *Stub) IsUnary() bool {
	// For v4 stubs, use method registry if available
	if len(s.InputsV4) > 0 {
		// This is a v4 stub, check method registry
		// Note: We need access to the budgerigar to check the registry
		// For now, fall back to legacy logic
		return len(s.Inputs) == 0
	}

	// Legacy logic: unary if no Inputs
	return len(s.Inputs) == 0
}

// IsClientStream returns true if this stub is for client streaming requests.
// For v4 stubs, this is determined by the method registry.
// For legacy stubs, this is determined by having Inputs data.
func (s *Stub) IsClientStream() bool {
	// For v4 stubs, use method registry if available
	if len(s.InputsV4) > 0 {
		// This is a v4 stub, check method registry
		// Note: We need access to the budgerigar to check the registry
		// For now, fall back to legacy logic
		return len(s.Inputs) > 0
	}

	// Legacy logic: client stream if has Inputs
	return len(s.Inputs) > 0
}

// IsServerStream returns true if this stub is for server streaming responses.
// For v4 stubs, this is determined by the method registry.
// For legacy stubs, this is determined by having Output.Stream data.
func (s *Stub) IsServerStream() bool {
	// For v4 stubs, use method registry if available
	if len(s.OutputsRawV4) > 0 {
		// This is a v4 stub, check method registry
		// Note: We need access to the budgerigar to check the registry
		// For now, fall back to legacy logic
		return len(s.Output.Stream) > 0
	}

	// Legacy logic: server stream if has Output.Stream
	return len(s.Output.Stream) > 0
}

// IsBidirectional returns true if this stub can handle bidirectional streaming.
// For v4 stubs, this is determined by the method registry.
// For legacy stubs, this is determined by having both Inputs and Output.Stream data.
func (s *Stub) IsBidirectional() bool {
	// For v4 stubs, use method registry if available
	if len(s.InputsV4) > 0 || len(s.OutputsRawV4) > 0 {
		// This is a v4 stub, check method registry
		// Note: We need access to the budgerigar to check the registry
		// For now, fall back to legacy logic
		return s.IsClientStream() && s.IsServerStream()
	}

	// Legacy logic: bidirectional if both client and server stream
	return s.IsClientStream() && s.IsServerStream()
}

// Key returns the unique identifier of the stub.
func (s *Stub) Key() uuid.UUID {
	return s.ID
}

// Left returns the service name of the stub.
func (s *Stub) Left() string {
	return s.Service
}

// Right returns the method name of the stub.
func (s *Stub) Right() string {
	return s.Method
}

// Score returns the priority score of the stub.
func (s *Stub) Score() int {
	return s.Priority
}

// InputData represents the input data of a gRPC request.
type InputData struct {
	IgnoreArrayOrder bool           `json:"ignoreArrayOrder,omitempty"` // Whether to ignore the order of arrays in the input data.
	Equals           map[string]any `json:"equals"`                     // The data to match exactly.
	Contains         map[string]any `json:"contains"`                   // The data to match partially.
	Matches          map[string]any `json:"matches"`                    // The data to match using regular expressions.
	Any              []InputData    `json:"any"`                        // Logical OR group of matchers
}

// GetEquals returns the data to match exactly.
func (i InputData) GetEquals() map[string]any {
	return i.Equals
}

// GetContains returns the data to match partially.
func (i InputData) GetContains() map[string]any {
	return i.Contains
}

// GetMatches returns the data to match using regular expressions.
func (i InputData) GetMatches() map[string]any {
	return i.Matches
}

// InputHeader represents the headers of a gRPC request.
type InputHeader struct {
	Equals   map[string]any `json:"equals"`   // The headers to match exactly.
	Contains map[string]any `json:"contains"` // The headers to match partially.
	Matches  map[string]any `json:"matches"`  // The headers to match using regular expressions.
}

// GetEquals returns the headers to match exactly.
func (i InputHeader) GetEquals() map[string]any {
	return i.Equals
}

// GetContains returns the headers to match partially.
func (i InputHeader) GetContains() map[string]any {
	return i.Contains
}

// GetMatches returns the headers to match using regular expressions.
func (i InputHeader) GetMatches() map[string]any {
	return i.Matches
}

// Len returns the total number of headers to match.
func (i InputHeader) Len() int {
	return len(i.Equals) + len(i.Matches) + len(i.Contains)
}

// Output represents the output data of a gRPC response.
type Output struct {
	Headers map[string]string `json:"headers"`          // The headers of the response.
	Data    map[string]any    `json:"data,omitempty"`   // The data of the response.
	Stream  []any             `json:"stream,omitempty"` // The stream data for server-side streaming.
	// Each element represents a message to be sent.
	Error string           `json:"error"`           // The error message of the response.
	Code  *codes.Code      `json:"code,omitempty"`  // The status code of the response.
	Delay gptypes.Duration `json:"delay,omitempty"` // The delay of the response or error.
}

// HasTemplates checks if the output contains any template strings.
func (o *Output) HasTemplates() bool {
	return (o.Data != nil && template.HasTemplates(o.Data)) ||
		(o.Stream != nil && template.HasTemplatesInStream(o.Stream)) ||
		(o.Error != "" && template.IsTemplateString(o.Error)) ||
		(o.Headers != nil && template.HasTemplatesInHeaders(o.Headers))
}

// ProcessDynamicOutput processes the output data and applies dynamic templates.
func (o *Output) ProcessDynamicOutput(
	requestData map[string]any,
	headers map[string]any,
	messageIndex int,
	allMessages []any,
	attemptNumber int,
	maxAttempts int,
	stubID string,
) error {
	engine := template.New()

	// Create template data
	templateData := template.Data{
		Request:       requestData,
		Headers:       headers,
		MessageIndex:  messageIndex,
		RequestTime:   time.Now(),
		Timestamp:     time.Now(),
		State:         make(map[string]any),
		Requests:      allMessages,
		AttemptNumber: attemptNumber,
		AttemptIndex:  attemptNumber,
		MaxAttempts:   maxAttempts,
		TotalAttempts: maxAttempts,
		StubID:        stubID,
		RequestID:     stubID,
	}

	// Process all output fields
	if o.Data != nil {
		err := engine.ProcessMap(o.Data, templateData)
		if err != nil {
			return err
		}
	}

	if o.Stream != nil {
		err := engine.ProcessStream(o.Stream, templateData)
		if err != nil {
			return err
		}
	}

	if o.Error != "" {
		renderedError, err := engine.ProcessError(o.Error, templateData)
		if err != nil {
			return err
		}

		if renderedError != "" {
			o.Error = renderedError
		}
	}

	if o.Headers != nil {
		err := engine.ProcessHeaders(o.Headers, templateData)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetOutputsV4 returns the typed v4 outputs if available.
func (s *Stub) GetOutputsV4() []OutputV4 {
	return s.OutputsV4
}

// HasOutputsV4 returns true if the stub has v4 outputs.
func (s *Stub) HasOutputsV4() bool {
	return len(s.OutputsV4) > 0
}

// IsV4Stub returns true if this is a v4 format stub.
func (s *Stub) IsV4Stub() bool {
	return len(s.InputsV4) > 0 || len(s.OutputsRawV4) > 0 || len(s.OutputsV4) > 0
}

// GetFirstDataOutput returns the first data output from v4 outputs.
func (s *Stub) GetFirstDataOutput() *types.DataRule {
	if len(s.OutputsV4) == 0 {
		return nil
	}

	for _, output := range s.OutputsV4 {
		if output.Data != nil {
			return output.Data
		}
	}

	return nil
}

// GetFirstStreamOutput returns the first stream output from v4 outputs.
func (s *Stub) GetFirstStreamOutput() *types.StreamRule {
	if len(s.OutputsV4) == 0 {
		return nil
	}

	for _, output := range s.OutputsV4 {
		if output.Stream != nil {
			return output.Stream
		}
	}

	return nil
}

// GetFirstSequenceOutput returns the first sequence output from v4 outputs.
func (s *Stub) GetFirstSequenceOutput() *types.SequenceRule {
	if len(s.OutputsV4) == 0 {
		return nil
	}

	for _, output := range s.OutputsV4 {
		if output.Sequence != nil {
			return output.Sequence
		}
	}

	return nil
}
