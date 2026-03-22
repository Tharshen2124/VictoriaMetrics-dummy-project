// Package handlers contains the HTTP handler functions for the e-commerce API.
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"tharshen2124/vivmdummyproject/db"
	"tharshen2124/vivmdummyproject/models"
	"tharshen2124/vivmdummyproject/utils"
)

// createUserRequest is the expected JSON body for POST /api/users.
type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CreateUser handles POST /api/users.
func CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("[HANDLER] CreateUser: invalid JSON body: %v\n", err)
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)

	if req.Name == "" || req.Email == "" {
		fmt.Printf("[HANDLER] CreateUser: missing required fields name=%q email=%q\n", req.Name, req.Email)
		utils.Error(w, http.StatusBadRequest, "name and email are required")
		return
	}

	exists, err := db.Store.EmailExists(req.Email)
	if err != nil {
		fmt.Printf("[HANDLER] CreateUser: EmailExists error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if exists {
		fmt.Printf("[HANDLER] CreateUser: duplicate email %q\n", req.Email)
		utils.Error(w, http.StatusConflict, "a user with that email already exists")
		return
	}

	user := models.User{
		ID:        utils.NewID(),
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}

	created, err := db.Store.CreateUser(user)
	if err != nil {
		fmt.Printf("[HANDLER] CreateUser: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.JSON(w, http.StatusCreated, created)
}

// GetUser handles GET /api/users/{id}.
func GetUser(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	user, err := db.Store.GetUser(id)
	if errors.Is(err, db.ErrNotFound) {
		fmt.Printf("[HANDLER] GetUser: user id=%s not found\n", id)
		utils.Error(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		fmt.Printf("[HANDLER] GetUser: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.JSON(w, http.StatusOK, user)
}
