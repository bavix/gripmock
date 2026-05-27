package proto

import (
	"strings"

	protosetdom "github.com/bavix/gripmock/v3/internal/domain/protoset"
)

// ParseArgumentsWithBindings parses raw command-line arguments to extract per-proxy source bindings.
// It detects -S flags before proxy URLs and binds them to that specific proxy.
//
// The cmdSources parameter contains -S flags collected by Cobra's flag system.
// When no per-proxy bindings are found, all sources (cmdSources + positional) are used globally.
//
// Examples:
//
//	-S a.proto -S b.proto grpc+proxy://up1:4111 grpc+proxy://up2:4222
//	  → up1:4111 gets [a.proto, b.proto], up2:4222 uses reflection
//
//	-S a.proto grpc+proxy://up1:4111 -S b.proto grpc+proxy://up2:4222
//	  → up1:4111 gets [a.proto], up2:4222 gets [b.proto]
//
//	grpc+proxy://up1:4111 -S a.proto grpc+proxy://up2:4222
//	  → up1:4111 uses reflection, up2:4222 gets [a.proto]
func ParseArgumentsWithBindings(args []string, imports []string, cmdSources []string) *Arguments { //nolint:cyclop
	var protoPath []string

	var pendingSources []string

	var bindings []ProxySourceBinding

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "-S" || arg == "--source" {
			if i+1 < len(args) {
				i++
				pendingSources = append(pendingSources, args[i])
			}

			continue
		}

		if value, ok := strings.CutPrefix(arg, "-S="); ok {
			pendingSources = append(pendingSources, value)

			continue
		}

		if value, ok := strings.CutPrefix(arg, "--source="); ok {
			pendingSources = append(pendingSources, value)

			continue
		}

		if IsProxyURL(arg) {
			bindings = append(bindings, ProxySourceBinding{
				ProxyURL: arg,
				Sources:  append([]string{}, pendingSources...),
			})
			pendingSources = nil

			continue
		}

		protoPath = append(protoPath, arg)
	}

	if len(bindings) == 0 {
		allSources := make([]string, 0, len(pendingSources)+len(cmdSources))
		allSources = append(allSources, cmdSources...)
		allSources = append(allSources, pendingSources...)

		return New(protoPath, imports, allSources)
	}

	if len(cmdSources) > 0 && len(bindings) > 0 && len(bindings[0].Sources) == 0 {
		bindings[0].Sources = append(cmdSources, bindings[0].Sources...)
	}

	return NewWithBindings(protoPath, imports, bindings)
}

// IsProxyURL checks if the argument is a proxy URL by attempting to parse it
// and checking if it has a ProxyMode set. This uses the canonical source parser
// from protoset package, ensuring consistency with the rest of the codebase.
func IsProxyURL(arg string) bool {
	source, err := protosetdom.ParseSource(arg)
	if err != nil {
		return false
	}

	return source.ProxyMode != ""
}
