package schema

type Database struct {
	SQL   *SQL   `yaml:"sql"`
	Redis *Redis `yaml:"redis"`
}

type SQL struct {
	Host              string             `yaml:"host"`
	Port              int                `yaml:"port"`
	Database          string             `yaml:"database"`
	Username          string             `yaml:"username"`
	Password          string             `yaml:"password"`
	PoolingConnection *PoolingConnection `yaml:"poolingConnection"`
}

type PoolingConnection struct {
	MaxIdle     int   `yaml:"maxIdle"`
	MaxOpen     int   `yaml:"maxOpen"`
	MaxLifetime int64 `yaml:"maxLifetime"`
}

type Redis struct {
	Mode string `yaml:"mode"`
	RedisCluster
}

type RedisSingle struct {
	Address         string  `yaml:"address"`
	Username        *string `yaml:"username"`
	Password        *string `yaml:"password"`
	DB              *int    `yaml:"db"`
	Network         *string `yaml:"network"`
	MaxRetries      *int    `yaml:"maxRetries"`
	MaxRetryBackoff *int    `yaml:"maxRetryBackoff"`
	MinRetryBackoff *int    `yaml:"minRetryBackoff"`
	DialTimeout     *int    `yaml:"dialTimeout"`
	ReadTimeout     *int    `yaml:"readTimeout"`
	WriteTimeout    *int    `yaml:"writeTimeout"`
	PoolFIFO        *bool   `yaml:"poolFIFO"`
	PoolSize        *int    `yaml:"poolSize"`
	PoolTimeout     *int    `yaml:"poolTimeout"`
	MinIdleConns    *int    `yaml:"minIdleConns"`
	MaxIdleConns    *int    `yaml:"maxIdleConns"`
}

type RedisCluster struct {
	RedisSingle
	SentinelAddress         []string `yaml:"sentinelAddress"`
	MasterName              string   `yaml:"masterName"`
	RouteByLatency          *bool    `yaml:"routeByLatency"`
	RouteRandomly           *bool    `yaml:"routeRandomly"`
	ReplicaOnly             *bool    `yaml:"replicaOnly"`
	UseDisconnectedReplicas *bool    `yaml:"useDisconnectedReplicas"`
}
