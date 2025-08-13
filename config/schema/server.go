package schema

type Server struct {
	RestAPI RestAPI `yaml:"restApi"`
}

type RestAPI struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}
