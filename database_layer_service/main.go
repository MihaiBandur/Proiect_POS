package main

import (
	"log"

	"job-management-service/internal/auth"
	"job-management-service/internal/config"
	"job-management-service/internal/handler"
	"job-management-service/internal/repository/memory"
)

func main() {
	cfg := config.Load()

	validator, err := auth.NewGRPCValidator(cfg.AuthGRPCAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer validator.Close()

	// Stocare in memorie (fazele 2-4); in Faza 5 o inlocuim cu MongoDB.
	router := handler.NewRouter(handler.Dependencies{
		Validator: validator,
		Accounts:  validator, // acelasi client gRPC face si ValidateToken, si ListUsers
		Tx:        memory.NewTransactor(),
		Scripts:   memory.NewScriptRepository(),
		Users:     memory.NewUserRepository(),
		Computers: memory.NewComputerRepository(),
		Clusters:  memory.NewClusterRepository(),
	})

	log.Printf("Job Management Service asculta pe portul %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
