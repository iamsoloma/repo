package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Config struct {
	User     string
	Password string
	Domain   string
	Port     string
	DBName   string
}

type Database struct {
	conn *pgx.Conn
}

func GetConnectionString(config Config) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		config.User,
		config.Password,
		config.Domain,
		config.Port,
		config.DBName)
}

func NewPostgresConnection(config Config) (*Database, error) {
	conn, err := pgx.Connect(context.Background(), GetConnectionString(config))
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
	if err := db.migrateRepositories(); err != nil {
		return err
	}
	return nil
}
