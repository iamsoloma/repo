package main

import (
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v6/osfs"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/backend"
	"github.com/go-git/go-git/v6/plumbing/transport"
)

// Credentials - простое хранилище учётных данных
var credentials = map[string]string{
	"user":   "password",
	"admin":  "admin123",
	"soloma": "test",
}

// AuthenticatedBackendHandler обёртка над backend с аутентификацией
type AuthenticatedBackendHandler struct {
	backend *backend.Backend
	logger  *log.Logger
}

// NewAuthenticatedBackendHandler создаёт новый обработчик с аутентификацией
func NewAuthenticatedBackendHandler(back *backend.Backend, logger *log.Logger) *AuthenticatedBackendHandler {
	return &AuthenticatedBackendHandler{
		backend: back,
		logger:  logger,
	}
}

// validateBasicAuth проверяет Basic Authentication
func (h *AuthenticatedBackendHandler) validateBasicAuth(r *http.Request) (bool, string) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false, "missing authorization header"
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Basic" {
		return false, "invalid authorization header format"
	}

	credBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false, "invalid base64 encoding"
	}

	userPass := strings.SplitN(string(credBytes), ":", 2)
	if len(userPass) != 2 {
		return false, "invalid credentials format"
	}

	username, password := userPass[0], userPass[1]

	// Проверяем учётные данные
	storedPassword, exists := credentials[username]
	if !exists || storedPassword != password {
		return false, "invalid username or password"
	}

	return true, username
}

// ServeHTTP реализует http.Handler с проверкой аутентификации
func (h *AuthenticatedBackendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Логируем запрос
	h.logger.Printf("[%s] %s %s %s", r.Method, r.URL.Path, r.RemoteAddr, r.Header.Get("Authorization"))

	service := r.URL.Query().Get("service")
	if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/info/refs") && service == transport.ReceivePackService {
		if strings.TrimSpace(r.Header.Get("Authorization")) == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Git Repository"`)
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			h.logger.Printf("AUTH CHALLENGE - %s", r.URL.Path)
			return
		}
	}

	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git-receive-pack") {
		isAuthed, user := h.validateBasicAuth(r)
		if !isAuthed {
			w.Header().Set("WWW-Authenticate", `Basic realm="Git Repository"`)
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			h.logger.Printf("AUTH FAILED - %s", r.URL.Path)
			return
		}
		h.logger.Printf("AUTH SUCCESS - user: %s", user)
	}

	// Передаём запрос на обработку backend'ом
	h.backend.ServeHTTP(w, r)
}

func ensureRepo(base, repo string) error {
	repoPath := filepath.Join(base, repo)
	var err error
	if _, err = os.Stat(repoPath); err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	if err = os.MkdirAll(repoPath, 0o755); err != nil {
		return err
	}

	_, err = git.PlainInit(repoPath, true)
	return err
}

func main() {
	logger := log.New(
		log.Writer(),
		"[GIT-SERVER] ",
		log.LstdFlags|log.Lshortfile,
	)

	repoBase := filepath.Join(".", "repos")
	repoName := filepath.Join("soloma", "test.git")
	if err := ensureRepo(repoBase, repoName); err != nil {
		logger.Fatal("failed to initialize repository:", err)
	}

	loader := transport.NewFilesystemLoader(osfs.New(repoBase), false)
	back := backend.New(loader)

	// Обёртываем backend с аутентификацией
	handler := NewAuthenticatedBackendHandler(back, logger)

	http.Handle("/", handler)

	addr := ":3000"
	logger.Printf("Git HTTP Server started at http://localhost%s", addr)
	logger.Printf("Repository path: %s", filepath.Join(repoBase, repoName))
	logger.Printf("Available credentials:")
	for user, pass := range credentials {
		logger.Printf("  - %s:%s", user, pass)
	}

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		logger.Fatal(err)
	}
}
