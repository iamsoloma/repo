package storage

import "context"

type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	CreatedAt    string
}

func (db *Database) migrateUsers() error {
	_, err := db.conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(255) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func (db *Database) CreateUser(user *User) (err error, user_id int64) {
	err = db.conn.QueryRow(context.Background(), `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id;
	`, user.Username, user.Email, user.PasswordHash).Scan(&user.ID)
	return err, user.ID
}

func (db *Database) GetUserByID(userID int64) (*User, error) {
	var user User
	err := db.conn.QueryRow(context.Background(), `
		SELECT id, username, email, password_hash, created_at FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt)

	return &user, err
}

func (db *Database) GetUserByUsername(username string) (*User, error) {
	var user User
	err := db.conn.QueryRow(context.Background(), `
		SELECT id, username, email, password_hash, created_at FROM users WHERE username = $1
	`, username).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt)

	return &user, err
}