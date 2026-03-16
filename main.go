package main

import (
	"log"
	"net/http"
	"strings"

	"repo/db"
	"repo/git"
	"repo/handlers"
)

func main() {
	if err := db.Init("./gitbox.db"); err != nil {
		log.Fatal("db init:", err)
	}
	log.Println("Database initialized")

	if err := handlers.Init("./templates"); err != nil {
		log.Fatal("templates init:", err)
	}
	log.Println("Templates loaded")

	// Git server — auth is handled by our middleware wrapper.
	gitServer := git.New(git.Config{
		Dir:        "./repos",
		AutoCreate: false,
		AutoHooks:  false,
		Auth:       false,
	})

	if err := gitServer.Setup(); err != nil {
		log.Fatal("git setup:", err)
	}

	mux := http.NewServeMux()

	// Static assets
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Everything else
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path

		if isGitRequest(p) {
			serveGit(w, r, gitServer)
			return
		}

		switch p {
		case "/":
			handlers.Home(w, r)
		case "/login":
			if r.Method == http.MethodPost {
				handlers.LoginPOST(w, r)
			} else {
				handlers.LoginGET(w, r)
			}
		case "/register":
			if r.Method == http.MethodPost {
				handlers.RegisterPOST(w, r)
			} else {
				handlers.RegisterGET(w, r)
			}
		case "/logout":
			handlers.Logout(w, r)
		case "/dashboard":
			handlers.Dashboard(w, r)
		case "/new":
			if r.Method == http.MethodPost {
				handlers.NewRepoPOST(w, r)
			} else {
				handlers.NewRepoGET(w, r)
			}
		default:
			routeDynamic(w, r)
		}
	})

	log.Println("Starting GitBox on :5000")
	if err := http.ListenAndServe("0.0.0.0:5000", mux); err != nil {
		log.Fatal(err)
	}
}

// isGitRequest returns true for git smart-HTTP protocol paths.
func isGitRequest(p string) bool {
	return strings.HasSuffix(p, "/info/refs") ||
		strings.HasSuffix(p, "/git-upload-pack") ||
		strings.HasSuffix(p, "/git-receive-pack")
}

// serveGit authenticates git requests and proxies them to the git server.
// Public repos allow unauthenticated clones; private repos and all pushes
// require valid DB credentials. Only the repo owner may push.
func serveGit(w http.ResponseWriter, r *http.Request, srv *git.Server) {
	owner, repo := parseGitPath(r.URL.Path)
	if owner == "" || repo == "" {
		http.NotFound(w, r)
		return
	}

	repoRecord, err := db.GetRepository(owner, repo)
	if err != nil || repoRecord == nil {
		http.NotFound(w, r)
		return
	}

	isPush := strings.HasSuffix(r.URL.Path, "/git-receive-pack") ||
		(strings.HasSuffix(r.URL.Path, "/info/refs") &&
			r.URL.Query().Get("service") == "git-receive-pack")

	// Public read: allow without credentials.
	if !repoRecord.IsPrivate && !isPush {
		if r.Header.Get("Authorization") == "" {
			srv.ServeHTTP(w, r)
			return
		}
	}

	// Require authentication.
	user, pass, ok := r.BasicAuth()
	if !ok || user == "" {
		w.Header().Set("WWW-Authenticate", `Basic realm="GitBox"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	authUser, err := db.AuthenticateUser(user, pass)
	if err != nil || authUser == nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="GitBox"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Private repo: only owner can read.
	if repoRecord.IsPrivate && authUser.Username != owner {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Push: only owner can write.
	if isPush && authUser.Username != owner {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	srv.ServeHTTP(w, r)
}

// parseGitPath extracts owner and repo name from a git smart-HTTP path.
// E.g. /alice/myrepo.git/info/refs  →  ("alice", "myrepo")
func parseGitPath(p string) (owner, repo string) {
	for _, suffix := range []string{"/info/refs", "/git-upload-pack", "/git-receive-pack"} {
		if strings.HasSuffix(p, suffix) {
			p = strings.TrimSuffix(p, suffix)
			break
		}
	}
	p = strings.TrimPrefix(p, "/")
	parts := strings.SplitN(p, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git")
}

// routeDynamic handles /:username and /:username/:repo paths.
func routeDynamic(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.SplitN(p, "/", 3)
	switch len(parts) {
	case 1:
		if parts[0] != "" {
			handlers.UserProfile(w, r, parts[0])
		} else {
			http.NotFound(w, r)
		}
	case 2:
		handlers.RepoView(w, r, parts[0], strings.TrimSuffix(parts[1], ".git"))
	default:
		http.NotFound(w, r)
	}
}
