package schema

type Config struct {
	App           App           `mapstructure:"app"`
	Database      Database      `mapstructure:"database"`
	Log           Log           `mapstructure:"log"`
	Otel          Otel          `mapstructure:"otel"`
	Server        Server        `mapstructure:"server"`
	Storage       Storage       `mapstructure:"storage"`
	MessageBroker MessageBroker `yaml:"messageBroker"`
}
