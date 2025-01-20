package model

type Otel struct {
	Trace  *otelTrace  `mapstructure:"trace"`
	Metric *otelMetric `mapstructure:"metric"`
}

type otelTrace struct {
	Exporters *otelExporters `mapstructure:"exporters"`
}

type otelMetric struct {
	InstrumentationName string         `mapstructure:"instrumentationName"`
	Exporters           *otelExporters `mapstructure:"exporters"`
}

type otelExporters struct {
	Otlp   *OtelExportersOtlp `mapstructure:"otlp"`
	Enable bool               `mapstructure:"enable"`
}

type OtelExportersOtlp struct {
	Address                     string `mapstructure:"address"`
	Timeout                     int    `mapstructure:"timeout"`
	ClientMaxReceiveMessageSize string `mapstructure:"clientMaxReceiveMessageSize"`
}
