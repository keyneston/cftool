package config

import "os"

const (
	DefaultCacheDir = "~/.cftool/cache"
	DefaultConfig   = "~/.cftool/config.yml"
	EnvVariable     = "CFTOOLRC"
)

func FindConfig() string {
	env := os.Getenv(EnvVariable)
	if env != "" {
		return env
	}

	return DefaultConfig
}
