package main

import (
	"log"
	"net/http"

	"github.com/sosedoff/gitkit"
)

func main() {
	// Configure git hooks
	hooks := &gitkit.HookScripts{
		PreReceive: `echo "Hello World!"`,
		PostReceive: `echo "Hello World!" > file.txt`,
	}

	// Configure git service
	service := gitkit.New(gitkit.Config{
		Dir:        "./repos",
		AutoCreate: true,
		AutoHooks:  true,
		Auth:       true,
		Hooks:      hooks,
	})

	service.AuthFunc = func(c gitkit.Credential, r *gitkit.Request) (bool, error) {
		log.Println("Auth: ", c.Username, c.Password, r.RepoName)
		return c.Username == "soloma", nil
	}

	// Configure git server. Will create git repos path if it does not exist.
	// If hooks are set, it will also update all repos with new version of hook scripts.
	if err := service.Setup(); err != nil {
		log.Fatal(err)
	}

	http.Handle("/", service)

	// Start HTTP server
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal(err)
	}
}
