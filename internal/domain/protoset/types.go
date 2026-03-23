package protoset

type SourceType int

const (
	SourceUnknown SourceType = iota
	SourceBufBuild
	SourceProto
	SourceDescriptor
	SourceDirectory
)

type Source struct {
	Type    SourceType
	Raw     string
	Module  string
	Version string
	Path    string
}
