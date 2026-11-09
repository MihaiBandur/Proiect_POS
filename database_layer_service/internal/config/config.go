package config

import (
	"os"
)

type Config struct {
	Port         string
	AuthGRPCAddr string
}

func Load() Config {
	return Config{
		Port:         GetEnv("PORT", "8080"),
		AuthGRPCAddr: GetEnv("AUTH_GRPC_ADDR", "127.0.0.1:50051"),
	}
}

func GetEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
