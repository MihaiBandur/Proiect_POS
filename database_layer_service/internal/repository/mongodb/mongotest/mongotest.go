package mongotest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"job-management-service/internal/config"
	"job-management-service/internal/repository/mongodb"
)

func Enabled() bool {
	return os.Getenv("MONGO_TESTS") == "1"
}

func Config() (config.MongoConfig, error) {
	_, file, _, _ := runtime.Caller(0)
	_ = godotenv.Load(filepath.Join(filepath.Dir(file), "../../../../../.env"))
	cfg := config.Load().Mongo
	devAddr := cfg.Addr
	cfg.Addr = config.GetEnv("MONGO_TEST_ADDR", "127.0.0.1:27018")
	if cfg.Addr == devAddr {
		return cfg, fmt.Errorf("MONGO_TEST_ADDR (%s) este baza de dezvoltare; refuz sa o golesc", cfg.Addr)
	}
	return cfg, nil

}

var (
	once   sync.Once
	shared *mongo.Database
	errDB  error
)

func Open() *mongo.Database {
	if !Enabled() {
		return nil
	}
	once.Do(func() {
		cfg, err := Config()
		if err != nil {
			errDB = err
			return
		}
		shared, errDB = mongodb.Connect(context.Background(), cfg)
	})
	if errDB != nil {
		panic(fmt.Sprintf("MongoDB de test: %v (ai pornit docker-compose --profile test up -d mongodb-test?)", errDB))
	}
	for _, name := range mongodb.CollectionNames() {
		if _, err := shared.Collection(name).DeleteMany(context.Background(), bson.D{}); err != nil {
			panic(fmt.Sprintf("golirea colectiei %s: %v", name, err))
		}
	}
	return shared
}

func Require(t testing.TB) *mongo.Database {
	t.Helper()
	if !Enabled() {
		t.Skip("ruleaza cu MONGO_TESTS=1 si containerul de test pornit")
	}
	return Open()
}
