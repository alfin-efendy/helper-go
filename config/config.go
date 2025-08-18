package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/alfin-efendy/helper-go/config/schema"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

var (
	Data *schema.Config
	raw  map[string]interface{}
)

func Load() {
	var filePath string
	flag.StringVar(&filePath, "config", "config.yml", "Path to the configuration file")
	flag.Parse()

	if err := unmarshal(filePath); err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}
}

func unmarshal(filePath string) (err error) {
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

	// store the raw config for later use
	raw = viperConfig.AllSettings()

	return err
}

func getVal(key string, config map[string]interface{}) interface{} {
	if key == "" {
		return nil
	}

	// split the key by dot
	keys := strings.SplitN(key, ".", 2)

	// if the key is not nested
	if v, ok := config[keys[0]]; ok {
		switch v := v.(type) {
		// if the value is a map, then it's nested
		case map[string]interface{}:
			return getVal(keys[1], v)
		default:
			return v
		}
	}
	return nil
}

// GetString use dot to get value from nested key
// ex: sql.host
func GetValue(key string) string {
	value := getVal(key, raw)
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
