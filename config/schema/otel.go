package schema

type Otel struct {
	Address string `yaml:"address"`
	Timeout int    `yaml:"timeout"`
	Trace   bool   `yaml:"trace"`
	Metric  bool   `yaml:"metric"`
}
