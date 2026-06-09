package main

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/go-git/go-billy/v6/osfs"
	"github.com/go-git/go-git/v6/backend"
	"github.com/go-git/go-git/v6/plumbing/transport"

	"repo/gogit"
)

func main() {
	logger := log.New(
		log.Writer(),
		"[GIT-SERVER] ",
		log.LstdFlags|log.Lshortfile,
	)

	repoBase := filepath.Join(".", "repos")

	loader := transport.NewFilesystemLoader(osfs.New(repoBase), false)
	back := backend.New(loader)

	// Обёртываем backend с аутентификацией
	handler := gogit.NewServer(back, func(gogit.Credential, *gogit.Request) (bool, error) {
		return true, nil
	})

	http.Handle("/", handler)

	addr := ":3000"
	logger.Printf("Git HTTP Server started at http://localhost%s", addr)
	logger.Printf("Auth: any credentials will be accepted (for testing purposes)")

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		logger.Fatal(err)
	}
}
