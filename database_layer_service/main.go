package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"job-management-service/internal/auth"
	"job-management-service/internal/config"
	"job-management-service/internal/executor"
	"job-management-service/internal/files"
	"job-management-service/internal/handler"
	"job-management-service/internal/repository/memory"
	"job-management-service/internal/repository/mongodb"
)

func main() {
	_ = godotenv.Load("../.env")
	cfg := config.Load()

	validator, err := auth.NewGRPCValidator(cfg.AuthGRPCAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer validator.Close()

	scriptFiles, err := files.NewScriptStore(cfg.ScriptsDir)
	if err != nil {
		log.Fatal(err)
	}

	// SP 1.3: la pornire verificam canalul catre executor si link-ul catre Job Status Service.
	dispatcher, mockReason := executor.New(cfg.IPCSocketPath, cfg.JobStatusURL)
	if dispatcher.Mock() {
		log.Printf("Task-uri in modul mock: %s", mockReason)
	} else {
		go waitForExecutor(dispatcher, cfg.IPCSocketPath)
	}

	deps := handler.Dependencies{
		// Fisierele scripturilor incarcate prin upload (SCRIPTS_DIR).
		ScriptFiles: scriptFiles,
		Executor:    dispatcher,
		Validator:   validator,
		Accounts:    validator,
		// Originile aplicatiilor web care pot apela API-ul din browser (CORS_ALLOWED_ORIGINS).
		AllowedOrigins: cfg.AllowedOrigins,
	}

	switch cfg.Storage {
	case "memory":
		log.Print("Stocare: memorie (datele se pierd la oprire)")

		deps.Tx = memory.NewTransactor()
		deps.Scripts = memory.NewScriptRepository()
		deps.Users = memory.NewUserRepository()
		deps.Computers = memory.NewComputerRepository()
		deps.Clusters = memory.NewClusterRepository()
		deps.Tasks = memory.NewTaskRepository()
	case "mongo":
		db, err := mongodb.Connect(context.Background(), cfg.Mongo)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Client().Disconnect(context.Background())
		log.Printf("Stocare: MongoDB, baza %s de la %s", db.Name(), cfg.Mongo.Addr)

		deps.Tx = mongodb.NewTransactor(db.Client())
		deps.Scripts = mongodb.NewScriptRepository(db)
		deps.Users = mongodb.NewUserRepository(db)
		deps.Computers = mongodb.NewComputerRepository(db)
		deps.Clusters = mongodb.NewClusterRepository(db)
		deps.Tasks = mongodb.NewTaskRepository(db)
	default:
		log.Fatalf("STORAGE trebuie sa fie memory sau mongo, nu %q", cfg.Storage)
	}

	router := handler.NewRouter(deps)

	log.Printf("Job Management Service asculta pe portul %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

// waitForExecutor verifica in fundal ca executorul raspunde. In containerul din student-kit el
// porneste odata cu serviciul si asteapta intai MongoDB, deci ii dam timp; serviciul nu se
// blocheaza intre timp (task-urile raman pending si executorul le gaseste la scanare).
func waitForExecutor(dispatcher executor.Dispatcher, socketPath string) {
	for range 30 {
		if err := dispatcher.Health(context.Background()); err == nil {
			log.Printf("Executorul raspunde pe %s", socketPath)
			return
		}
		time.Sleep(2 * time.Second)
	}
	log.Printf("Executorul nu raspunde pe %s; task-urile raman pending pana porneste", socketPath)
}
