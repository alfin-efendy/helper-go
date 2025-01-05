package model

type app struct {
	Name   string `mapstructure:"name"`
	Mode   string `mapstructure:"mode"`
	Domain string `mapstructure:"domain"`
}
