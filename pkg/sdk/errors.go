package sdk

import "github.com/cockroachdb/errors"

var (
	ErrDescriptorsRequired                = errors.New("gripmock: descriptors required (use WithDescriptors or MockFrom)")
	ErrNoUsableServicesFoundViaReflection = errors.New("no services found via reflection (or only grpc.reflection/grpc.health)")
	ErrUnexpectedResponse                 = errors.New("unexpected response: not ListServicesResponse")
	ErrVerificationFailed                 = errors.New("gripmock: expectations not met")
	ErrInvalidInput                       = errors.New("gripmock: invalid input")
	ErrReflection                         = errors.New("gripmock: reflection error")
)

// ExpectationNotMetError describes a single unmet expectation for ExpectationsWereMet.
// Is(ErrVerificationFailed) returns true so callers can use errors.Is.
type ExpectationNotMetError struct {
	Service  string
	Method   string
	Expected int
	Actual   int
}

func (e *ExpectationNotMetError) Error() string {
	return "gripmock: " + e.Service + "/" + e.Method +
		": expected " + itoa(e.Expected) + " call(s), got " + itoa(e.Actual)
}

func (e *ExpectationNotMetError) Is(target error) bool {
	return target == ErrVerificationFailed
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var buf [20]byte

	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}

	return string(buf[pos:])
}
