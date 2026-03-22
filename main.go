package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"tharshen2124/vivmdummyproject/config"
	"tharshen2124/vivmdummyproject/db"
	"tharshen2124/vivmdummyproject/routes"
)

func main() {
	// Load configuration from .env file.
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("[SERVER] Failed to load config: %v\n", err)
		return
	}

	// Connect to the Postgres database.
	if err := db.InitDB(cfg.DatabaseURL); err != nil {
		fmt.Printf("[SERVER] Failed to connect to database: %v\n", err)
		return
	}

	// Build and configure the router.
	r := mux.NewRouter()
	routes.RegisterRoutes(r)

	addr := ":" + cfg.Port
	fmt.Printf("[SERVER] Starting on port %s\n", cfg.Port)

	if err := http.ListenAndServe(addr, r); err != nil {
		fmt.Printf("[SERVER] ListenAndServe error: %v\n", err)
	}
}
