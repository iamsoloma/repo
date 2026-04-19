package storage

import (
	"context"
	"time"
)

type User struct {
	Username     string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

func (db *Database) migrateUsers() error {
	_, err := db.conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS users (
			username VARCHAR(255) PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func (db *Database) CreateUser(user *User) (err error) {
	_, err = db.conn.Exec(context.Background(), `
		INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3)
	`, user.Username, user.Email, user.PasswordHash)
	return err
}

func (db *Database) GetUserByUsername(username string) (*User, error) {
	var user User
	err := db.conn.QueryRow(context.Background(), `
		SELECT username, email, password_hash, created_at FROM users WHERE username = $1
	`, username).Scan(&user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt)

	return &user, err
}
