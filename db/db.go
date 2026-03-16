package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type Repository struct {
	ID          int64
	OwnerID     int64
	OwnerName   string
	Name        string
	Description string
	IsPrivate   bool
	CreatedAt   time.Time
}

var DB *sql.DB

func Init(path string) error {
	var err error
	DB, err = sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	return migrate()
}

func migrate() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			username      TEXT    UNIQUE NOT NULL,
			email         TEXT    UNIQUE NOT NULL,
			password_hash TEXT    NOT NULL,
			created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS repositories (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			owner_id    INTEGER NOT NULL,
			name        TEXT    NOT NULL,
			description TEXT    NOT NULL DEFAULT '',
			is_private  INTEGER NOT NULL DEFAULT 0,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (owner_id) REFERENCES users(id),
			UNIQUE(owner_id, name)
		);

		CREATE TABLE IF NOT EXISTS sessions (
			token      TEXT    PRIMARY KEY,
			user_id    INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
	`)
	return err
}

func CreateUser(username, email, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	res, err := DB.Exec(
		`INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`,
		username, email, string(hash),
	)
	if err != nil {
		return nil, err
	}

	id, _ := res.LastInsertId()
	return &User{ID: id, Username: username, Email: email}, nil
}

func GetUserByUsername(username string) (*User, error) {
	u := &User{}
	err := DB.QueryRow(
		`SELECT id, username, email, password_hash, created_at FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func GetUserByID(id int64) (*User, error) {
	u := &User{}
	err := DB.QueryRow(
		`SELECT id, username, email, password_hash, created_at FROM users WHERE id = ?`,
		id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func AuthenticateUser(username, password string) (*User, error) {
	u, err := GetUserByUsername(username)
	if err != nil || u == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return u, nil
}

func CreateSession(userID int64) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	_, err := DB.Exec(`INSERT INTO sessions (token, user_id) VALUES (?, ?)`, token, userID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func GetSessionUser(token string) (*User, error) {
	var userID int64
	err := DB.QueryRow(`SELECT user_id FROM sessions WHERE token = ?`, token).Scan(&userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return GetUserByID(userID)
}

func DeleteSession(token string) error {
	_, err := DB.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func CreateRepository(ownerID int64, name, description string, isPrivate bool) (*Repository, error) {
	priv := 0
	if isPrivate {
		priv = 1
	}
	res, err := DB.Exec(
		`INSERT INTO repositories (owner_id, name, description, is_private) VALUES (?, ?, ?, ?)`,
		ownerID, name, description, priv,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Repository{
		ID:          id,
		OwnerID:     ownerID,
		Name:        name,
		Description: description,
		IsPrivate:   isPrivate,
	}, nil
}

func GetRepository(ownerName, repoName string) (*Repository, error) {
	r := &Repository{}
	var priv int
	err := DB.QueryRow(`
		SELECT r.id, r.owner_id, u.username, r.name, r.description, r.is_private, r.created_at
		FROM repositories r
		JOIN users u ON u.id = r.owner_id
		WHERE u.username = ? AND r.name = ?`,
		ownerName, repoName,
	).Scan(&r.ID, &r.OwnerID, &r.OwnerName, &r.Name, &r.Description, &priv, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.IsPrivate = priv == 1
	return r, nil
}

func ListUserRepositories(username string) ([]Repository, error) {
	rows, err := DB.Query(`
		SELECT r.id, r.owner_id, u.username, r.name, r.description, r.is_private, r.created_at
		FROM repositories r
		JOIN users u ON u.id = r.owner_id
		WHERE u.username = ?
		ORDER BY r.created_at DESC`,
		username,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		var r Repository
		var priv int
		if err := rows.Scan(&r.ID, &r.OwnerID, &r.OwnerName, &r.Name, &r.Description, &priv, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.IsPrivate = priv == 1
		repos = append(repos, r)
	}
	return repos, nil
}

func ListPublicRepositories() ([]Repository, error) {
	rows, err := DB.Query(`
		SELECT r.id, r.owner_id, u.username, r.name, r.description, r.is_private, r.created_at
		FROM repositories r
		JOIN users u ON u.id = r.owner_id
		WHERE r.is_private = 0
		ORDER BY r.created_at DESC
		LIMIT 50`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		var r Repository
		var priv int
		if err := rows.Scan(&r.ID, &r.OwnerID, &r.OwnerName, &r.Name, &r.Description, &priv, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.IsPrivate = priv == 1
		repos = append(repos, r)
	}
	return repos, nil
}
