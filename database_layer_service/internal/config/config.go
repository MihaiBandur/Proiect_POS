package config

import (
	"os"
	"strings"
)

type Config struct {
	Port         string
	AuthGRPCAddr string

	Storage string
	Mongo   MongoConfig

	AllowedOrigins []string
	// ScriptsDir: folderul fisierelor de scripturi. In Docker va fi volumul partajat (SP 1.3).
	ScriptsDir string
	// IPCSocketPath si JobStatusURL: socketul executorului din student-kit si link-ul catre
	// Job Status Service. Daca lipseste oricare, serviciul raspunde cu task-uri mock (SP 1.3).
	IPCSocketPath string
	JobStatusURL  string
}

type MongoConfig struct {
	Addr     string
	Database string
	Username string
	Password string
}

func Load() Config {
	return Config{
		Port:           GetEnv("PORT", "8080"),
		AuthGRPCAddr:   GetEnv("AUTH_GRPC_ADDR", "127.0.0.1:50051"),
		Storage:        GetEnv("STORAGE", "memory"),
		AllowedOrigins: splitList(GetEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
		// Local, un folder din database_layer_service (ignorat de git); in Docker, volumul partajat.
		ScriptsDir:    GetEnv("SCRIPTS_DIR", "data/scripts"),
		IPCSocketPath: os.Getenv("IPC_SOCKET_PATH"),
		JobStatusURL:  os.Getenv("JOB_STATUS_URL"),

		Mongo: MongoConfig{
			Addr:     GetEnv("MONGO_ADDR", "127.0.0.1:27017"),
			Database: os.Getenv("MONGO_DB_NAME"),
			Username: os.Getenv("MONGO_APP_USERNAME"),
			Password: os.Getenv("MONGO_APP_PASSWORD"),
		},
	}
}

func GetEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func splitList(value string) []string {
	var items []string
	for item := range strings.SplitSeq(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}
