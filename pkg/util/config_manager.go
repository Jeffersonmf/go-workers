package util

import (
	"errors"
	"io/fs"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// A missing .env is not an error: configuration may come entirely
	// from real environment variables (the common case in a
	// container). Anything else reading the file (permissions,
	// malformed content) is worth knowing about, so only the
	// not-found case is swallowed.
	//
	// viper.ConfigFileNotFoundError is what ReadInConfig returns when a
	// config *name* (SetConfigName + AddConfigPath) was searched for
	// and not found in any path. SetConfigFile above sets an explicit
	// path instead, so a missing file surfaces as the plain OS error
	// from opening it — checking for the wrong error type here silently
	// never matched, and every missing-.env startup logged a warning
	// that was meant to be filtered out.
	if err := viper.ReadInConfig(); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			Sugar.Warnf("could not read .env: %v", err)
		}
		return
	}

	watchConfig()
}

// ReadParameter returns the value of a configuration key, sourced from
// .env if present and overridden by an actual environment variable of
// the same name, or "" if it is not set.
//
// The original implementation was `fmt.Sprint(viper.Get(parameter))`:
// fmt.Sprint on a nil `any` (exactly what viper.Get returns for an
// unset key) prints the four-character string "<nil>", not "". Every
// caller checking ReadParameter(key) != "" to mean "is it set" was
// silently wrong for every unset key. viper.GetString does the right
// type coercion, including for an unset key, directly.
func ReadParameter(parameter string) string {
	return viper.GetString(parameter)
}

func watchConfig() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		if err := viper.ReadInConfig(); err != nil {
			Sugar.Warnf("config file changed but failed to reload: %v", err)
			return
		}
		Sugar.Infof("config file changed: %s", e.Name)
	})
}
