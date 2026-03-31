package storage

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Config struct {
	ConnectionString string
}

type Database struct {
	conn *pgx.Conn
}

func NewPostgresConnection(config Config) (*Database, error) {
	conn, err := pgx.Connect(context.Background(), config.ConnectionString)
	if err != nil {
		return nil, err
	}
	return &Database{conn: conn}, nil
}

func (db *Database) Close() error {
	return db.conn.Close(context.Background())
}

func (db *Database) GetPostgresVersion() (string, error) {
	var version string
	err := db.conn.QueryRow(context.Background(), "SELECT version()").Scan(&version)
	if err != nil {
		return "", err
	}
	return version, nil
}

// Migrate call migration`s functions for all entities
func (db *Database) Migrate() error {
	if err := db.migrateUsers(); err != nil {
		return err
	}
	return nil
}