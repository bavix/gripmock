package proto

type Arguments struct {
	protoPath []string
	imports   []string
}

func New(protoPath []string, imports []string) *Arguments {
	return &Arguments{
		protoPath: protoPath,
		imports:   imports,
	}
}

func (p *Arguments) ProtoPath() []string {
	return p.protoPath
}

func (p *Arguments) Imports() []string {
	return p.imports
}
