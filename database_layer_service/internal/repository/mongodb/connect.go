package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"job-management-service/internal/config"
)

const connectTimeout = 5 * time.Second

func Connect(ctx context.Context, cfg config.MongoConfig) (*mongo.Database, error) {
	if missing := missingSettings(cfg); len(missing) > 0 {
		return nil, fmt.Errorf("lipsesc variabilele de mediu %s", strings.Join(missing, ", "))
	}

	opts := options.Client().
		// directConnection: imaginea atlas-local ruleaza ca replica set cu un singur nod. Fara
		// aceasta optiune, driverul ar incerca sa ajunga la nod dupa numele lui din interiorul
		// containerului, care nu exista pe calculatorul tau.
		ApplyURI("mongodb://" + cfg.Addr + "/?directConnection=true").
		// Credentialele separat, nu in adresa: un caracter special in parola (@, :, /)
		// ar strica adresa. Utilizatorul e definit in baza proiectului (01-create-users.js).
		SetAuth(options.Credential{
			Username:   cfg.Username,
			Password:   cfg.Password,
			AuthSource: cfg.Database,
		}).
		// O lista nil din Go se scrie [] in loc de null: schema din 02-schema.js refuza null.
		SetBSONOptions(&options.BSONOptions{NilSliceAsEmpty: true}).
		// Daca baza nu raspunde, o operatie renunta dupa atat (implicit ar astepta 30 s);
		// la 5.4 transformam eroarea in 503.
		SetServerSelectionTimeout(3 * time.Second).SetTimeout(5 * time.Second)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("configurarea clientului MongoDB: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("MongoDB nu raspunde la %s sau autentificarea a esuat: %w", cfg.Addr, err)
	}
	return client.Database(cfg.Database), nil
}

func missingSettings(cfg config.MongoConfig) []string {
	var missing []string
	if cfg.Database == "" {
		missing = append(missing, "MONGO_DB_NAME")
	}
	if cfg.Username == "" {
		missing = append(missing, "MONGO_APP_USERNAME")
	}
	if cfg.Password == "" {
		missing = append(missing, "MONGO_APP_PASSWORD")
	}
	return missing
}
