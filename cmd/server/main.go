package main

import (
	"log"

	"github.com/sumanthd032/codedrop/internal/db"
	"github.com/sumanthd032/codedrop/internal/store"
)

func main() {
	// Database
	database, err := db.NewConnection()
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	if err := database.Migrate(); err != nil {
		log.Fatalf("Could not apply migrations: %v", err)
	}

	// Storage (S3/MinIO)
	s3Store, err := store.NewS3Store()
	if err != nil {
		log.Fatalf("Could not connect to storage: %v", err)
	}
	
	// Test: Upload a tiny test file
	err = s3Store.UploadChunk("test-connection.txt", []byte("Hello CodeDrop Storage!"))
	if err != nil {
		log.Fatalf("Storage test failed: %v", err)
	}
	log.Println("Successfully connected to S3/MinIO & uploaded test file!")

	log.Println("CodeDrop Server is ready...")
}