package model

type otel struct {
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
	Otlp   *otelExportersOtlp `mapstructure:"otlp"`
	Enable bool               `mapstructure:"enable"`
}

type otelExportersOtlp struct {
	Address                     string `mapstructure:"address"`
	Timeout                     int    `mapstructure:"timeout"`
	ClientMaxReceiveMessageSize string `mapstructure:"clientMaxReceiveMessageSize"`
}
