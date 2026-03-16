# Git HTTP Server

A Go-based HTTP git server that exposes git repositories over HTTP with optional authentication and git hook support.

## Overview

This project implements a smart HTTP git server in Go. It allows clients to clone, push, and pull git repositories over HTTP.

## Architecture

- **Language**: Go 1.21
- **Entry point**: `main.go`
- **Git package**: `git/` — contains the HTTP handler, SSH server, hook management, and auth logic
- **Port**: 5000 (HTTP)

## Key Features

- Smart HTTP git protocol support
- Authentication via custom `AuthFunc` callback
- Auto-create repositories on first push
- Server-side git hooks (pre-receive, update, post-receive)
- SSH support (`git/ssh.go`)

## Configuration (main.go)

- `Dir`: `./repos` — where repositories are stored
- `AutoCreate`: `true` — auto-creates repos on push
- `AutoHooks`: `true` — installs hook scripts on setup
- `Auth`: `true` — requires authentication
- Auth is currently hardcoded to accept username `soloma` with any password

## Development

Run the server:
```
go run .
```

## Dependencies

- `github.com/gofrs/uuid` — UUID generation
- `golang.org/x/crypto` — SSH support
- `github.com/stretchr/testify` — testing

## Workflow

- **Start application**: Runs `go run .` on port 5000
