package main

import (
	"log"

	"repo/server"
	"repo/storage"
)

func main() {

	srv, err := server.NewServer(server.Config{
		ListenAddr: ":3000",
		Storage: &storage.Config{
			User:     "main",
			Password: "qwerty",
			Domain:   "localhost",
			Port:     "5432",
			DBName:   "repo",
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
