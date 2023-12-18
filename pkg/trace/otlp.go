package trace

type OTLPTrace struct {
	Endpoint    string  `envconfig:"OTLP_TRACE_GRPC_ENDPOINT"`
	TLS         bool    `envconfig:"OTLP_TRACE_TLS" default:"false"`
	SampleRatio float64 `envconfig:"OTLP_SAMPLE_RATIO"`
}

func (o *OTLPTrace) UseTrace() bool {
	return o.Endpoint != "" && o.SampleRatio > 0
}
