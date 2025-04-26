package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
)

//it`s really work!
func main() {
	reposRoot := os.Getenv("GIT_PROJECT_ROOT")
	if reposRoot == "" {
		log.Fatal("GIT_PROJECT_ROOT env variable is not set")
	}

	gitPath, err := exec.LookPath("git")
	if err != nil {
		log.Fatalf("cannot find git: %v", err)
	}
	log.Printf("using a git at: %q", gitPath)

	gitHandler := &cgi.Handler{
		Path: gitPath,
		Args: []string{"http-backend"},
		Env: []string{
			fmt.Sprintf("=%s", reposRoot),
			"GIT_HTTP_EXPORT_ALL=true",
		},
	}

	log.Fatal(http.ListenAndServe("0.0.0.0:8080", gitHandler))

}
