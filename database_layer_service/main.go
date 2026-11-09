package main

import (
	"log"

	"job-management-service/internal/auth"
	"job-management-service/internal/config"
	"job-management-service/internal/handler"
)

func main() {
	cfg := config.Load()

	validator, err := auth.NewGRPCValidator(cfg.AuthGRPCAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer validator.Close()

	router := handler.NewRouter(validator)

	log.Printf("Job Management Service asculta pe portul %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
