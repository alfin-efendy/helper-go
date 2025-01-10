package model

type Config struct {
	App      app      `mapstructure:"app"`
	Database database `mapstructure:"database"`
	Log      log      `mapstructure:"log"`
	Otel     otel     `mapstructure:"otel"`
	Server   server   `mapstructure:"server"`
	Token    *token   `mapstructure:"token"`
	Storage  *storage `mapstructure:"storage"`
}
