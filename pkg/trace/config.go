package trace

type OTLPTrace struct {
	Host        string  `envconfig:"OTLP_TRACE_GRPC_HOST" default:"127.0.0.1"`
	Port        string  `envconfig:"OTLP_TRACE_GRPC_PORT" default:"4317"`
	TLS         bool    `envconfig:"OTLP_TRACE_TLS" default:"false"`
	SampleRatio float64 `envconfig:"OTLP_SAMPLE_RATIO"`
}

func (o *OTLPTrace) UseTrace() bool {
	return o.Host != "" && o.Port != "" && o.SampleRatio > 0
}
