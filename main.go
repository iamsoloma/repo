package main

import (
	"log"
	"net/http"

	"repo/git"
	"repo/server"
	"repo/storage"
)

func main() {
	// Configure git hooks
	hooks := &git.HookScripts{
		PreReceive:  `echo "Hello World!"`,
		PostReceive: `echo "Hello World!" > file.txt`,
	}

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

	// Configure git service
	service := git.New(git.Config{
		Dir:        "./repos",
		AutoCreate: true,
		AutoHooks:  true,
		Auth:       true,
		Hooks:      hooks,
	})

	service.AuthFunc = func(c git.Credential, r *git.Request) (bool, error) {
		log.Println("Auth: ", c.Username, c.Password, r.RepoName)
		_, err := srv.DB.GetUserByUsername(c.Username)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	// Configure git server. Will create git repos path if it does not exist.
	// If hooks are set, it will also update all repos with new version of hook scripts.
	if err := service.Setup(); err != nil {
		log.Fatal(err)
	}

	http.Handle("/", service)

	// Start HTTP server
	log.Printf("Server started at %s", srv.Config.ListenAddr)
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal(err)
	}
}
