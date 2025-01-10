package model

type storage struct {
	Driver        *string `mapstructure:"driver"`
	Endpoint      string  `mapstructure:"endpoint"`
	AccessKey     string  `mapstructure:"accessKey"`
	SecretKey     string  `mapstructure:"secretKey"`
	BucketName    string  `mapstructure:"bucketName"`
	UseSSL        bool    `mapstructure:"useSSL"`
	RetentionDays int     `mapstructure:"retentionDays"`
}
