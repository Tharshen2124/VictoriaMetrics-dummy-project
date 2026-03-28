// Package handlers contains the HTTP handler functions for the e-commerce API.
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
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

	requestCount, _   = meter.Int64Counter("http.request.count",
		metric.WithDescription("Total number of HTTP requests"),
	)

	requestDuration, _ = meter.Float64Histogram("http.request.duration",
		metric.WithDescription("HTTP request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	
	userCreatedCount, _ = meter.Int64Counter("user.created.count",
		metric.WithDescription("Total number of users created"),
	)

	errorCount, _ = meter.Int64Counter("http.error.count",
		metric.WithDescription("Total number of HTTP errors"),
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
	start := time.Now()

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

		requestCount.Add(r.Context(), 1, metric.WithAttributes(
			attribute.String("handler", "CreateUser"),
			attribute.Int("status", http.StatusBadRequest),
		))
		errorCount.Add(r.Context(), 1, metric.WithAttributes(
			attribute.String("handler", "CreateUser"),
			attribute.String("error_type", "invalid_json"),
		))
		requestDuration.Record(r.Context(), float64(time.Since(start).Milliseconds()), metric.WithAttributes(
			attribute.String("handler", "CreateUser"),
		))

		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	logFields["user_email"] = req.Email
	logFields["user_name"] = req.Name

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

	logFields["user_id"] = user.ID
	logFields["status"] = http.StatusCreated
	logger.InfoContext(r.Context(), "user created successfully", mapToSlogAttrs(logFields)...)

	requestCount.Add(r.Context(), 1, metric.WithAttributes(
		attribute.String("handler", "CreateUser"),
		attribute.Int("status", http.StatusCreated),
	))
	
	userCreatedCount.Add(r.Context(), 1)
	
	requestDuration.Record(r.Context(), float64(time.Since(start).Milliseconds()), metric.WithAttributes(
		attribute.String("handler", "CreateUser"),
	))

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
