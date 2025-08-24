package runtime

import (
	"context"
	"time"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

// Command interface for stream operations.
type Command interface {
	Execute(ctx context.Context, w Writer) error
}

// SendCommand represents a send operation.
type SendCommand struct {
	data    map[string]any
	headers map[string]string
}

// NewSendCommand creates a new send command.
func NewSendCommand(data map[string]any, headers map[string]string) *SendCommand {
	return &SendCommand{
		data:    data,
		headers: headers,
	}
}

// Execute sends data through the writer.
func (s *SendCommand) Execute(ctx context.Context, w Writer) error {
	return w.Send(applyRuntimeTemplates(s.data))
}

// DelayCommand represents a delay operation.
type DelayCommand struct {
	duration time.Duration
}

// NewDelayCommand creates a new delay command.
func NewDelayCommand(duration time.Duration) *DelayCommand {
	return &DelayCommand{
		duration: duration,
	}
}

// Execute waits for the specified duration.
func (d *DelayCommand) Execute(ctx context.Context, w Writer) error {
	if d.duration > 0 {
		time.Sleep(d.duration)
	}

	return nil
}

// EndCommand represents an end operation.
type EndCommand struct {
	status *domain.GrpcStatus
}

// NewEndCommand creates a new end command.
func NewEndCommand(status *domain.GrpcStatus) *EndCommand {
	return &EndCommand{
		status: status,
	}
}

// Execute ends the stream with status.
func (e *EndCommand) Execute(ctx context.Context, w Writer) error {
	return w.End(e.status)
}

// CompositeCommand represents a sequence of commands.
type CompositeCommand struct {
	commands []Command
}

// NewCompositeCommand creates a new composite command.
func NewCompositeCommand(commands ...Command) *CompositeCommand {
	return &CompositeCommand{
		commands: commands,
	}
}

// Execute executes all commands in sequence.
func (c *CompositeCommand) Execute(ctx context.Context, w Writer) error {
	for _, cmd := range c.commands {
		if err := cmd.Execute(ctx, w); err != nil {
			return err
		}
	}

	return nil
}

// CommandFactory creates commands from stream steps.
type CommandFactory struct{}

// NewCommandFactory creates a new command factory.
func NewCommandFactory() *CommandFactory {
	return &CommandFactory{}
}

// CreateCommands creates commands from a stream step.
func (cf *CommandFactory) CreateCommands(step domain.StreamStepStrict) []Command {
	var commands []Command

	// Add send command if present
	if step.Send != nil {
		commands = append(commands, NewSendCommand(step.Send.Data, step.Send.Headers))
	}

	// Add delay command if present
	if step.Delay != nil {
		if d, err := globalParser.parseDuration(step.Delay.Duration); err == nil {
			commands = append(commands, NewDelayCommand(d))
		}
	}

	// Add end command if present
	if step.End != nil {
		status := &domain.GrpcStatus{
			Code:    step.End.Code,
			Message: step.End.Message,
		}
		commands = append(commands, NewEndCommand(status))
	}

	return commands
}
