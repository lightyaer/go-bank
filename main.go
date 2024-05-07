package main

import (
	"log"
)

func main() {

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
