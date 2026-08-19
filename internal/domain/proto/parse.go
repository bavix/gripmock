package proto

import (
	"slices"
	"strings"

	protosetdom "github.com/bavix/gripmock/v3/internal/domain/protoset"
)

// ParseArgumentsWithBindings extracts per-proxy source bindings from the command line.
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
func ParseArgumentsWithBindings(positional, rawArgs, imports, cmdSources []string) *Arguments {
	bindings, trailingSources, sawSourceFlag := scanBindings(rawArgs)
	protoPath := protoPathFrom(positional)

	if len(bindings) == 0 {
		// cmdSources and the sources scanned off the line are the same -S flags
		// seen from two angles: Cobra collected them, and the scan found them
		// again. Concatenating would build every source twice.
		return New(protoPath, imports, mergeSources(cmdSources, trailingSources))
	}

	if !sawSourceFlag && len(cmdSources) > 0 && len(bindings[0].Sources) == 0 {
		bindings[0].Sources = append(cmdSources, bindings[0].Sources...)
	}

	return NewWithBindings(protoPath, imports, bindings)
}

func mergeSources(cmdSources, scanned []string) []string {
	merged := make([]string, 0, len(cmdSources)+len(scanned))
	seen := make(map[string]struct{}, len(cmdSources)+len(scanned))

	for _, source := range slices.Concat(cmdSources, scanned) {
		if _, duplicate := seen[source]; duplicate {
			continue
		}

		seen[source] = struct{}{}

		merged = append(merged, source)
	}

	if len(merged) == 0 {
		return nil
	}

	return merged
}

func scanBindings(args []string) ([]ProxySourceBinding, []string, bool) {
	var (
		bindings       []ProxySourceBinding
		pendingSources []string
		sawSourceFlag  bool
	)

	for i := range args {
		arg := args[i]

		if source, ok := sourceFlagValue(args, i); ok {
			sawSourceFlag = true

			if source != "" {
				pendingSources = append(pendingSources, source)
			}

			continue
		}

		if IsProxyURL(arg) {
			bindings = append(bindings, ProxySourceBinding{
				ProxyURL: arg,
				Sources:  append([]string{}, pendingSources...),
			})
			pendingSources = nil
		}
	}

	return bindings, pendingSources, sawSourceFlag
}

func sourceFlagValue(args []string, i int) (string, bool) {
	if value, ok := inlineSourceValue(args[i]); ok {
		return value, true
	}

	if isBareSourceFlag(args[i]) {
		if i+1 < len(args) {
			return args[i+1], true
		}

		return "", true
	}

	if i > 0 && isBareSourceFlag(args[i-1]) {
		return "", true
	}

	return "", false
}

func isBareSourceFlag(arg string) bool {
	return arg == "-S" || arg == "--source"
}

func inlineSourceValue(arg string) (string, bool) {
	for _, prefix := range []string{"-S=", "--source=", "-S"} {
		if value, ok := strings.CutPrefix(arg, prefix); ok && value != "" {
			return value, true
		}
	}

	return "", false
}

func protoPathFrom(positional []string) []string {
	var protoPath []string

	for i := range positional {
		if _, isSource := sourceFlagValue(positional, i); isSource {
			continue
		}

		if IsProxyURL(positional[i]) {
			continue
		}

		protoPath = append(protoPath, positional[i])
	}

	return protoPath
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
