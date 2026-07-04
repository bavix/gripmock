package sdk

import (
	"errors"
	"fmt"
	"strings"
)

// ParseFullMethodName parses a gRPC full method name (for example
// "/helloworld.Greeter/SayHello") into service and method parts.
//
// It also accepts values without leading slash ("helloworld.Greeter/SayHello").
func ParseFullMethodName(fullMethod string) (string, string, error) {
	fullMethod = strings.TrimPrefix(strings.TrimSpace(fullMethod), "/")
	if fullMethod == "" {
		return "", "", errors.New("sdk: full method name is empty")
	}

	service, method, ok := strings.Cut(fullMethod, "/")
	if !ok || service == "" || method == "" || strings.Contains(method, "/") {
		return "", "", fmt.Errorf("sdk: invalid full method name %q", fullMethod)
	}

	return service, method, nil
}

// By parses full gRPC method name and returns (service, method).
//
// Example: By("/helloworld.Greeter/SayHello") returns
// ("helloworld.Greeter", "SayHello").
//
// Panics on invalid input. Use ParseFullMethodName for error-returning behavior.
func By(fullMethod string) (string, string) {
	service, method, err := ParseFullMethodName(fullMethod)
	if err != nil {
		panic(err)
	}

	return service, method
}
