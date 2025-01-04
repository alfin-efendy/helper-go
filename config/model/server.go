package model

type server struct {
	RestAPI *restAPI `mapstructure:"restAPI"`
}

type restAPI struct {
	Port   int   `mapstructure:"port"`
	Stdout bool  `mapstructure:"stdout"`
	Cors   *cors `mapstructure:"cors"`
}

type cors struct {
	AllowOrigins     []string `mapstructure:"allowOrigins"`
	AllowMethods     []string `mapstructure:"allowMethods"`
	AllowHeaders     []string `mapstructure:"allowHeaders"`
	AllowCredentials bool     `mapstructure:"allowCredentials"`
	ExposeHeaders    []string `mapstructure:"exposeHeaders"`
	MaxAge           int      `mapstructure:"maxAge"`
}
