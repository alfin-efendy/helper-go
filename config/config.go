package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/alfin-efendy/helper-go/config/model"
	"github.com/alfin-efendy/helper-go/utility"
	"github.com/spf13/viper"
)

var (
	Config      *model.Config
	ViperConfig *viper.Viper
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
}
