package server

import (
	"time"

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
