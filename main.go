package main

import "repo/api"

func main() {
	config := api.Config{
		ListenAddr:  "0.0.0.0:3000",
		ReposFolder: "./repos",
	}
	server := api.NewServer(config)
	server.Start()
}
