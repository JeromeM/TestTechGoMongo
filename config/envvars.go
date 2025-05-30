package config

import (
	"os"

	"github.com/kataras/golog"
)

func RequireEnvVar(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		golog.Fatalf("%s env var required!", key)
	}

	return value
}
