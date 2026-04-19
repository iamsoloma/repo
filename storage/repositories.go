package storage

import "context"

type Repository struct {
	ID          int64
	Name        string
	Owner       string
	Description string
	isPrivate   bool
}

func (db *Database) migrateRepositories() error {
	_, err := db.conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS repositories (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			owner VARCHAR(255) NOT NULL,
			description TEXT,
			is_private BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func (db *Database) CreateRepository(repo *Repository) (err error, repoID int64) {
	err = db.conn.QueryRow(context.Background(), `
		INSERT INTO repositories (name, owner, description, is_private)
		VALUES ($1, $2, $3, $4)
		RETURNING id;
	`, repo.Name, repo.Owner, repo.Description, repo.isPrivate).Scan(&repo.ID)
	return err, repo.ID
}

func (db *Database) GetRepositoryByID(repoID int64) (*Repository, error) {
	var repo Repository
	err := db.conn.QueryRow(context.Background(), `
		SELECT id, name, owner, description, is_private FROM repositories WHERE id = $1
	`, repoID).Scan(&repo.ID, &repo.Name, &repo.Owner, &repo.Description, &repo.isPrivate)

	return &repo, err
}

func (db *Database) GetRepositoryByName(name string) (*Repository, error) {
	var repo Repository
	err := db.conn.QueryRow(context.Background(), `
		SELECT id, name, owner, description, is_private FROM repositories WHERE name = $1
	`, name).Scan(&repo.ID, &repo.Name, &repo.Owner, &repo.Description, &repo.isPrivate)

	return &repo, err
}

func (db *Database) GetRepositoriesByOwner(owner string) ([]*Repository, error) {
	rows, err := db.conn.Query(context.Background(), `
		SELECT id, name, owner, description, is_private FROM repositories WHERE owner = $1
	`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repositories []*Repository
	for rows.Next() {
		var repo Repository
		if err := rows.Scan(&repo.ID, &repo.Name, &repo.Owner, &repo.Description, &repo.isPrivate); err != nil {
			return nil, err
		}
		repositories = append(repositories, &repo)
	}
	return repositories, rows.Err()
}
