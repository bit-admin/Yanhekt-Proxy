package config

import (
	"os"
	"time"
)

type Config struct {
	Port             string
	UpstreamAPI      string
	VideoHost        string
	MagicKey         string
	LogLevel         string
	RequestTimeout   time.Duration
	IntranetTimeout  time.Duration
	MappingsFile     string
}

func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		UpstreamAPI:      getEnv("UPSTREAM_API", "https://cbiz.yanhekt.cn"),
		VideoHost:        getEnv("VIDEO_HOST", "cvideo.yanhekt.cn"),
		MagicKey:         getEnv("MAGIC_KEY", "1138b69dfef641d9d7ba49137d2d4875"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		RequestTimeout:   parseDuration(getEnv("REQUEST_TIMEOUT", "30s"), 30*time.Second),
		IntranetTimeout:  parseDuration(getEnv("INTRANET_TIMEOUT", "8s"), 8*time.Second),
		MappingsFile:     getEnv("MAPPINGS_FILE", "./mappings.json"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string, defaultValue time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultValue
	}
	return d
}
