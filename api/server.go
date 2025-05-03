package api

import (
	"log/slog"
	"net/http"
	"time"
)

type Config struct {
	ListenAddr string
	ReposFolder string
}

type Server struct {
	*Config
	Started time.Time
}

func NewServer(config Config) (*Server) {
	return &Server{
		Config: &config,
	}
}

func (s *Server) Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /user/repos/{user}/{project}/{name}", s.CreateRepo)
	server := http.Server{
		Addr:    s.Config.ListenAddr,
		Handler: mux,
	}

	s.Started = time.Now().UTC()

	slog.Info("api is running", "address", s.Config.ListenAddr)
	err := server.ListenAndServe()
	if err != nil {
		slog.Error("API stoped", "error", err)
	}

}
