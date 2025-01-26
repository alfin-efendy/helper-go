package model

type Otel struct {
	Host    string `mapstructure:"host"`
	Timeout int    `mapstructure:"timeout"`
	Trace   bool   `mapstructure:"trace"`
	Metric  bool   `mapstructure:"metric"`
}
