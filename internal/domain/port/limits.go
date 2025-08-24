package port

import "context"

// LimitsRepository defines the interface for managing stub usage limits.
type LimitsRepository interface {
	// GetAttemptCount returns the current attempt count for a stub.
	GetAttemptCount(ctx context.Context, stubID string) int

	// IncrementAttempt increments the attempt count for a stub and returns the new count.
	IncrementAttempt(ctx context.Context, stubID string) int

	// ResetAttempts resets the attempt count for a stub.
	ResetAttempts(ctx context.Context, stubID string) error

	// IsLimitReached checks if the stub has reached its usage limit.
	IsLimitReached(ctx context.Context, stubID string, maxAttempts int) bool
}
