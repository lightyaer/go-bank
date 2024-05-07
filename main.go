package main

import (
	"log"

	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	store, err := NewPgStore()

	if err != nil {
		log.Fatal(err)
	}

	if err := store.Init(); err != nil {
		log.Fatal(err)
	}

	port := ":3000"
	server := NewAPIServer(port, store)
	server.Run()
}
