package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/alfin-efendy/helper-go/config/model"
	"github.com/alfin-efendy/helper-go/utility"
	"github.com/spf13/viper"
)

var Config *model.Config

func Load() {
	v := viper.New()
	v.AutomaticEnv()

	v.SetConfigName("config")
	v.AddConfigPath(".")

	err := v.ReadInConfig()
	if err != nil {
		utility.PrintPanic(fmt.Sprintf("Error reading config file: %s\n", err))
	}

	for _, k := range v.AllKeys() {
		value := v.GetString(k)
		if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
			v.Set(k, os.Getenv(strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")))
		}
	}

	Config = &model.Config{}

	if err = v.Unmarshal(Config); err != nil {
		utility.PrintPanic(fmt.Sprintf("Error reading config file: %s\n", err))
	}
}
