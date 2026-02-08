package main

import (
	"log"

	"github.com/sumanthd032/codedrop/internal/db"
)

func main() {
	// Initialize Database
	database, err := db.NewConnection()
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}

	// Run Migrations (Create tables)
	if err := database.Migrate(); err != nil {
		log.Fatalf("Could not apply migrations: %v", err)
	}

	log.Println("CodeDrop Server is ready to start...")
}