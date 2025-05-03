package api

import (
	"net/http"

	"github.com/go-git/go-git/v5"
)

func (s *Server) CreateRepo(w http.ResponseWriter, r *http.Request) {
	ownerName := r.PathValue("user")
	projectName := r.PathValue("project")
	repoName := r.PathValue("name")
	if repoName == "" || ownerName == "" || projectName == "" {
		http.Error(w, "Missing repository metadata", http.StatusBadRequest)
		return
	}

	// Create a new repository in the specified directory
	repoPath := s.Config.ReposFolder + "/" + repoName
	_, err := git.PlainInit(repoPath, true)
	if err != nil {
		http.Error(w, "Failed to create repository: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Repository created successfully"))
	return
}
