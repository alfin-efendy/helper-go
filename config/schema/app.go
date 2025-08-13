package schema

type App struct {
	Name string `mapstructure:"name"`
	Mode string `mapstructure:"mode"`
}
