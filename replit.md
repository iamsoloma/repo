# GitBox — Self-Hosted Git Server

A Go-based self-hosted git server with a full web UI. Users can register accounts, create public/private repositories, browse repos, and use git over HTTP.

## Architecture

- **Language**: Go 1.21
- **Port**: 5000 (HTTP)
- **Database**: SQLite via `modernc.org/sqlite` (pure Go, no CGo)
- **Entry point**: `main.go`

## Package Layout

| Package | Purpose |
|---------|---------|
| `git/` | Smart HTTP git protocol handler (clone/push/pull) |
| `db/` | SQLite database layer — users, repositories, sessions |
| `handlers/` | HTTP handlers for web UI pages |
| `templates/` | HTML templates (GitHub dark theme) |
| `static/` | CSS stylesheet |
| `repos/` | Bare git repositories on disk (`{owner}/{name}.git`) |

## Key Features

- **User accounts**: register, login, logout, sessions via HTTP-only cookie
- **Repositories**: create public/private repos; bare git repos initialized with `git init --bare`
- **Web UI**: home, dashboard, new repo, user profile, repo view pages
- **Git over HTTP**: smart HTTP protocol (clone, push, pull)
- **Auth rules**:
  - Public repos: unauthenticated clone allowed
  - Private repos: only owner can clone/push
  - Push: always requires credentials; only repo owner

## Database Schema

```sql
users        (id, username, email, password_hash, created_at)
repositories (id, owner_id, name, description, is_private, created_at)
sessions     (token, user_id, created_at)
```

## URL Routes

| Route | Description |
|-------|-------------|
| `GET /` | Home page (public repo listing) |
| `GET/POST /register` | User registration |
| `GET/POST /login` | Login |
| `POST /logout` | Logout |
| `GET /dashboard` | User's repository list |
| `GET/POST /new` | Create new repository |
| `GET /:user` | User profile |
| `GET /:user/:repo` | Repository page with clone URL |
| `GET /:user/:repo.git/info/refs` | Git smart HTTP (clone/fetch) |
| `POST /:user/:repo.git/git-receive-pack` | Git push |
| `POST /:user/:repo.git/git-upload-pack` | Git fetch |

## Configuration

- Git server: `Auth: false` (auth handled by middleware in `main.go`)
- Repos stored at: `./repos/{owner}/{name}.git`
- SQLite DB at: `./gitbox.db`
- Templates at: `./templates/`
- Static files at: `./static/`

## Dependencies

- `modernc.org/sqlite` — Pure-Go SQLite driver
- `golang.org/x/crypto` — bcrypt password hashing
- `github.com/gofrs/uuid` — UUID generation (git pkg)
- `github.com/stretchr/testify` — testing

## Development

```bash
go run .
```

## Workflow

- **Start application**: `go run .` on port 5000
