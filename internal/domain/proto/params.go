package proto

// ProxySourceBinding represents the binding between a proxy URL and its local descriptor sources.
type ProxySourceBinding struct {
	ProxyURL string
	Sources  []string
}

// Arguments represents the configuration for protobuf file processing.
type Arguments struct {
	protoPath     []string
	imports       []string
	sources       []string
	proxyBindings []ProxySourceBinding
}

// New creates a new Arguments instance with the specified protobuf paths, imports, and sources.
func New(protoPath []string, imports []string, sources []string) *Arguments {
	return &Arguments{
		protoPath:     ensureSlice(protoPath),
		imports:       ensureSlice(imports),
		sources:       ensureSlice(sources),
		proxyBindings: nil,
	}
}

// NewWithBindings creates a new Arguments instance with per-proxy source bindings.
func NewWithBindings(protoPath []string, imports []string, proxyBindings []ProxySourceBinding) *Arguments {
	// Deep copy bindings to prevent external modifications
	copiedBindings := make([]ProxySourceBinding, len(proxyBindings))
	for i, binding := range proxyBindings {
		copiedBindings[i] = ProxySourceBinding{
			ProxyURL: binding.ProxyURL,
			Sources:  ensureSlice(binding.Sources),
		}
	}

	return &Arguments{
		protoPath:     ensureSlice(protoPath),
		imports:       ensureSlice(imports),
		sources:       []string{}, // Always empty slice in binding mode
		proxyBindings: copiedBindings,
	}
}

// ensureSlice returns an empty slice if the input is nil, otherwise returns the input.
func ensureSlice(slice []string) []string {
	if slice == nil {
		return []string{}
	}

	return slice
}

// ProtoPath returns the list of protobuf file paths.
func (p *Arguments) ProtoPath() []string {
	return p.protoPath
}

// Imports returns the list of import paths for protobuf files.
func (p *Arguments) Imports() []string {
	return p.imports
}

// Sources returns the list of local descriptor sources for proxy modes (legacy global mode).
func (p *Arguments) Sources() []string {
	return p.sources
}

// ProxyBindings returns the per-proxy source bindings.
func (p *Arguments) ProxyBindings() []ProxySourceBinding {
	return p.proxyBindings
}

// HasProxyBindings returns true if per-proxy bindings are configured.
func (p *Arguments) HasProxyBindings() bool {
	return p.proxyBindings != nil
}
