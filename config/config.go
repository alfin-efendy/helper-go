package config

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/alfin-efendy/helper-go/config/schema"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

var Data *schema.Config

func Load() {
	var filePath string
	flag.StringVar(&filePath, "config", "config.yml", "Path to the configuration file")
	flag.Parse()

	if err := Unmarshal(filePath, &Data); err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}
}

func Unmarshal(filePath string, v interface{}) (err error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	viperConfig := viper.New()
	viperConfig.AutomaticEnv()
	viperConfig.SetConfigFile(filePath)

	err = viperConfig.ReadInConfig()
	if err != nil {
		return err
	}

	for _, k := range viperConfig.AllKeys() {
		value := viperConfig.GetString(k)
		if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
			viperConfig.Set(k, os.Getenv(strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")))
		}
	}

	err = viperConfig.Unmarshal(&Data)
	if err != nil {
		return err
	}

	if Data == nil {
		err = fmt.Errorf("config data is nil, please check your config file")
		return err
	}

	return err
}
