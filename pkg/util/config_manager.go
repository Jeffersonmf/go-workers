package util

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return
	}

	watcherConfig()
}

func ReadParameter(parameter string) string {
	return fmt.Sprint(viper.Get(parameter))
}

func watcherConfig() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		err := viper.ReadInConfig()
		if err != nil {
			Sugar.Infof("An error occurred")
		}

		Sugar.Infof("Config file changed:", e.Name)
	})
}
