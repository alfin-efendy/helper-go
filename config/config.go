package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/alfin-efendy/helper-go/config/model"
	"github.com/alfin-efendy/helper-go/utility"
	"github.com/spf13/viper"
)

var (
	Config *model.Config
	raw    map[string]interface{}
)

func Load() {
	ViperConfig := viper.New()
	ViperConfig.AutomaticEnv()

	ViperConfig.SetConfigName("config")
	ViperConfig.AddConfigPath(".")

	err := ViperConfig.ReadInConfig()
	if err != nil {
		utility.PrintPanic(fmt.Sprintf("Error reading config file: %s\n", err))
	}

	for _, k := range ViperConfig.AllKeys() {
		value := ViperConfig.GetString(k)
		if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
			ViperConfig.Set(k, os.Getenv(strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")))
		}
	}

	Config = &model.Config{}

	if err = ViperConfig.Unmarshal(Config); err != nil {
		utility.PrintPanic(fmt.Sprintf("Error reading config file: %s\n", err))
	}

	// store the raw config for later use
	raw = ViperConfig.AllSettings()
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
