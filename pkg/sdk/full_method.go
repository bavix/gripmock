package sdk

import (
	"fmt"
	"strings"
)

// ParseFullMethodName parses a gRPC full method name (for example
// "/helloworld.Greeter/SayHello") into service and method parts.
//
// It also accepts values without leading slash ("helloworld.Greeter/SayHello").
func ParseFullMethodName(fullMethod string) (service, method string, err error) {
	fullMethod = strings.TrimPrefix(strings.TrimSpace(fullMethod), "/")
	if fullMethod == "" {
		return "", "", fmt.Errorf("sdk: full method name is empty")
	}

	service, method, ok := strings.Cut(fullMethod, "/")
	if !ok || service == "" || method == "" || strings.Contains(method, "/") {
		return "", "", fmt.Errorf("sdk: invalid full method name %q", fullMethod)
	}

	return service, method, nil
}

// MustParseFullMethodName is ParseFullMethodName but panics on error.
func MustParseFullMethodName(fullMethod string) (service, method string) {
	service, method, err := ParseFullMethodName(fullMethod)
	panicIfErr(err)

	return service, method
}

// By parses full gRPC method name and returns (service, method).
//
// Example: By("/helloworld.Greeter/SayHello") returns
// ("helloworld.Greeter", "SayHello").
//
// Panics on invalid input. Use ParseFullMethodName for error-returning behavior.
func By(fullMethod string) (service, method string) {
	return MustParseFullMethodName(fullMethod)
}
