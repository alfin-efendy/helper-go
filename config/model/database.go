package model

type database struct {
	Sql   sql   `mapstructure:"sql"`
	Redis redis `mapstructure:"redis"`
}

type sql struct {
	Host              string             `mapstructure:"host"`
	Port              int                `mapstructure:"port"`
	Database          string             `mapstructure:"database"`
	Username          string             `mapstructure:"username"`
	Password          string             `mapstructure:"password"`
	PoolingConnection *poolingConnection `mapstructure:"poolingConnection"`
}

type poolingConnection struct {
	MaxIdle     int   `mapstructure:"maxIdle"`
	MaxOpen     int   `mapstructure:"maxOpen"`
	MaxLifetime int64 `mapstructure:"maxLifetime"`
}

type redis struct {
	Mode string `mapstructure:"mode"`
	redisCluster
}

type redisSingle struct {
	Address         string  `mapstructure:"address"`
	Username        *string `mapstructure:"username"`
	Password        *string `mapstructure:"password"`
	DB              *int    `mapstructure:"db"`
	Network         *string `mapstructure:"network"`
	MaxRetries      *int    `mapstructure:"maxRetries"`
	MaxRetryBackoff *int    `mapstructure:"maxRetryBackoff"`
	MinRetryBackoff *int    `mapstructure:"minRetryBackoff"`
	DialTimeout     *int    `mapstructure:"dialTimeout"`
	ReadTimeout     *int    `mapstructure:"readTimeout"`
	WriteTimeout    *int    `mapstructure:"writeTimeout"`
	PoolFIFO        *bool   `mapstructure:"poolFIFO"`
	PoolSize        *int    `mapstructure:"poolSize"`
	PoolTimeout     *int    `mapstructure:"poolTimeout"`
	MinIdleConns    *int    `mapstructure:"minIdleConns"`
	MaxIdleConns    *int    `mapstructure:"maxIdleConns"`
}

type redisCluster struct {
	redisSingle
	SentinelAddress         []string `mapstructure:"sentinelAddress"`
	MasterName              string   `mapstructure:"masterName"`
	RouteByLatency          *bool    `mapstructure:"routeByLatency"`
	RouteRandomly           *bool    `mapstructure:"routeRandomly"`
	ReplicaOnly             *bool    `mapstructure:"replicaOnly"`
	UseDisconnectedReplicas *bool    `mapstructure:"useDisconnectedReplicas"`
}
