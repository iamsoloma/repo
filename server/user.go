package server

import (
	"encoding/json"
	"log"
	"net/http"
	"repo/storage"
	"time"
)

type UserRequest struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"passwordHash"`
}

func (s *Server) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Println("Failed to decode user data: " + err.Error())
		return
	}

	u := storage.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: req.PasswordHash,
		CreatedAt:    time.Now().UTC(),
	}

	err := s.DB.CreateUser(&u)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		log.Println("Failed to create user: " + err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(u); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Println("Failed to encode user data: " + err.Error())
		return
	}
}

func (s *Server) GetUser(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	user, err := s.DB.GetUserByUsername(username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		log.Println("Failed to get user: " + err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Println("Failed to encode user data: " + err.Error())
		return
	}
}
