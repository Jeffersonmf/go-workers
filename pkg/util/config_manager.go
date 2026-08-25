package util

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

const envFilePath = ".env"

var (
	envFileMu   sync.RWMutex
	envFileVars = map[string]string{}
)

func init() {
	loadEnvFile()
}

// ReadParameter returns the value of a configuration key: a real
// environment variable takes precedence, then a value from .env, then
// "" if neither is set.
//
// This replaced spf13/viper, which pulled in roughly a dozen
// transitive dependencies (afero, cast, pflag, a TOML/HCL/INI parser,
// ...) to do what this package needs in about twenty lines: read
// KEY=VALUE lines from one file and fall back to real env vars. Viper
// supports config formats and sources (YAML, remote config stores,
// live directory watching) that this project never used.
func ReadParameter(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	envFileMu.RLock()
	defer envFileMu.RUnlock()
	return envFileVars[key]
}

func loadEnvFile() {
	file, err := os.Open(envFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			Logger.Warn("could not read .env", "error", err)
		}
		return
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			Logger.Warn("could not close .env", "error", cerr)
		}
	}()

	parsed := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		parsed[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	if err := scanner.Err(); err != nil {
		Logger.Warn("could not parse .env", "error", err)
		return
	}

	envFileMu.Lock()
	envFileVars = parsed
	envFileMu.Unlock()
}
