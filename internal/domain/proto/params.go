package proto

// ProtocParam represents the parameters for the protoc command.
type ProtocParam struct {
	// output is the output directory for the generated files.
	output string

	// protoPath is a list of paths to the proto files.
	protoPath []string

	// imports is a list of import paths.
	imports []string
}

func NewProtocParam(protoPath []string, output string, imports []string) *ProtocParam {
	return &ProtocParam{
		protoPath: protoPath,
		output:    output,
		imports:   imports,
	}
}

func (p *ProtocParam) ProtoPath() []string {
	return p.protoPath
}

func (p *ProtocParam) Imports() []string {
	return p.imports
}

func (p *ProtocParam) Output() string {
	return p.output
}
