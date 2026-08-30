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
	SourceProxy
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

	// Client certificate presented to an upstream that requires mTLS, plus the CA
	// that signs the upstream certificate.
	ReflectClientCert string
	ReflectClientKey  string
	ReflectCAFile     string
	ProxyMode         string
	RecordDelay       bool
}
