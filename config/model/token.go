package model

type token struct {
	AccessPrivateKey  string `mapstructure:"accessPrivateKey"`
	AccessPublicKey   string `mapstructure:"accessPublicKey"`
	AccessExpireHour  int    `mapstructure:"accessExpireHour"`
	RefreshPrivateKey string `mapstructure:"refreshPrivateKey"`
	RefreshPublicKey  string `mapstructure:"refreshPublicKey"`
	RefreshExpireHour int    `mapstructure:"refreshExpireHour"`
}
