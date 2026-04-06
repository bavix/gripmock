package protoset

import "time"

type SourceType int

const (
	SourceUnknown SourceType = iota
	SourceBufBuild
	SourceReflect
	SourceProto
	SourceDescriptor
	SourceDirectory
)

type Source struct {
	Type              SourceType
	Raw               string
	Module            string
	Version           string
	Path              string
	ReflectAddress    string
	ReflectTLS        bool
	ReflectServerName string
	ReflectBearer     string
	ReflectTimeout    time.Duration
	ReflectInsecure   bool
	ProxyMode         string
	RecordDelay       bool
}
