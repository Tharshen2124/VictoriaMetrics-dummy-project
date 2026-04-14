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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/metric"
	"tharshen2124/vivmdummyproject/db"
	"tharshen2124/vivmdummyproject/models"
	"tharshen2124/vivmdummyproject/utils"
)

const name = "vivm_dummy_project"

var (
	meter = otel.Meter(name)
	logger = otelslog.NewLogger(name)
	userCreatedCount, _ = meter.Int64Counter("user.created.count",
		metric.WithDescription("Total number of users created"),
	)
)

// createUserRequest is the expected JSON body for POST /api/users.
type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func mapToSlogAttrs(fields map[string]any) []any {
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	return attrs
}

// CreateUser handles POST /api/users.
func CreateUser(w http.ResponseWriter, r *http.Request) {

	logFields := map[string]any{
		"handler":  "CreateUser",
		"method":   r.Method,
		"path":     r.URL.Path,
	}

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logFields["error"] = err.Error()
		logFields["status"] = http.StatusBadRequest
		logger.ErrorContext(r.Context(), "invalid JSON body", mapToSlogAttrs(logFields)...)

		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	logFields["user"] = map[string]any{
		"name":  req.Name,
		"email": req.Email,
	}

	if req.Name == "" || req.Email == "" {
		logFields["error"] = "missing required fields"
		logFields["status"] = http.StatusBadRequest
		logger.WarnContext(r.Context(), "missing required fields", mapToSlogAttrs(logFields)...)
		utils.Error(w, http.StatusBadRequest, "name and email are required")
		return
	}

	exists, err := db.Store.EmailExists(req.Email)
	if err != nil {
		logFields["error"] = err.Error()
		logFields["status"] = http.StatusInternalServerError
		logger.ErrorContext(r.Context(), "email existence check failed", mapToSlogAttrs(logFields)...)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if exists {
		logFields["status"] = http.StatusConflict
		logger.WarnContext(r.Context(), "duplicate email", mapToSlogAttrs(logFields)...)
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
		logFields["error"] = err.Error()
		logFields["status"] = http.StatusInternalServerError
		logger.ErrorContext(r.Context(), "failed to create user", mapToSlogAttrs(logFields)...)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logFields["status"] = http.StatusCreated
	logger.InfoContext(r.Context(), "user created successfully", mapToSlogAttrs(logFields)...)
	
	userCreatedCount.Add(r.Context(), 1)
	utils.JSON(w, http.StatusCreated, created)
}

// GetUser handles GET /api/users/{id}.
func GetUser(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	logFields := map[string]any{
		"handler": "GetUser",
		"method":  r.Method,
		"path":    r.URL.Path,
		"user_id": id,
	}

	user, err := db.Store.GetUser(id)
	if errors.Is(err, db.ErrNotFound) {
		logFields["error"] = "user not found"
		logFields["status"] = http.StatusNotFound
		logger.ErrorContext(r.Context(), "user not found", mapToSlogAttrs(logFields)...)
		
		fmt.Printf("[HANDLER] GetUser: user id=%s not found\n", id)
		utils.Error(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		logFields["error"] = "db error"
		logFields["status"] = http.StatusInternalServerError
		logger.ErrorContext(r.Context(), "db error", mapToSlogAttrs(logFields)...)
		
		fmt.Printf("[HANDLER] GetUser: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logFields["status"] = http.StatusOK
	logger.InfoContext(r.Context(), "user retrieved successfully", mapToSlogAttrs(logFields)...)

	utils.JSON(w, http.StatusOK, user)
}
