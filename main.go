package main

import (
	"log"
)

func main() {
	store, err := NewPostgresStorage()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err := store.Init(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	server := NewAPIServer(":8080", store)
	server.Start()
}
