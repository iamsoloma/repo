package git

import (
	"fmt"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"time"
)

type Config struct {
	ListenAddr  string
	ReposFolder string
}

type Server struct {
	*Config
	Started time.Time
}

func NewServer(config Config) *Server {
	return &Server{
		Config: &config,
	}
}

func (s *Server) Start() error {
	//GIT_PROJECT_ROOT
	if s.Config.ReposFolder == "" {
		log.Fatal("REPOS_FOLDER is not set")
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
			fmt.Sprintf("=%s", s.Config.ReposFolder),
			"GIT_HTTP_EXPORT_ALL=true",
		},
		Stderr: os.Stderr,
		Logger: log.New(os.Stdout, "INFO", 0),
	}

	err = http.ListenAndServe(s.Config.ListenAddr, gitHandler)
	return err
}
