package server

import (
	"crypto/sha256"
	"encoding/base64"
	"log"
	"net/http"
	"time"

	"repo/git"
	"repo/storage"
)

type Config struct {
	ListenAddr string
	Storage    *storage.Config
}

type Server struct {
	Config    Config
	StartedAt time.Time
	DB        *storage.Database
}

func NewServer(config Config) (*Server, error) {

	db, err := storage.NewPostgresConnection(*config.Storage)
	if err != nil {
		return nil, err
	}

	err = db.Migrate()
	if err != nil {
		return nil, err
	}

	return &Server{
		Config: config,
		DB:     db,
	}, nil
}

func (s *Server) Start() error {
	s.StartedAt = time.Now()

	hooks := &git.HookScripts{
		PreReceive:  `echo "Hello From Repo!"`,
		PostReceive: `echo "Hello World!" > file.txt`,
	}

	gitService := git.New(git.Config{
		Dir:        "./repos",
		AutoCreate: true,
		AutoHooks:  true,
		Hooks:      hooks,
		Auth:       true,
	})

	gitService.AuthFunc = func(c git.Credential, r *git.Request) (bool, error) {
		log.Println("Auth: ", c.Username, c.Password, r.RepoName)
		u, err := s.DB.GetUserByUsername(c.Username)
		if err != nil {
			return false, err
		}
		h := sha256.New()
		h.Write([]byte(c.Password))
		if u.PasswordHash != base64.StdEncoding.EncodeToString((h.Sum(nil))) {
			return false, nil
		}
		return true, nil
	}

	if err := gitService.Setup(); err != nil {
		return err
	}

	http.Handle("/", gitService)

	log.Printf("Server started at %s", s.Config.ListenAddr)
	return http.ListenAndServe(s.Config.ListenAddr, nil)
}
