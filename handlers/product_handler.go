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

// createProductRequest is the expected JSON body for POST /api/products.
type createProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

// updateProductRequest is the expected JSON body for PUT /api/products/{id}.
// All fields are optional — only non-zero values overwrite the stored record.
type updateProductRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Price       *float64 `json:"price"`
	Stock       *int     `json:"stock"`
}

// handles GET /api/products.
func GetAllProducts(w http.ResponseWriter, r *http.Request) {
	products, err := db.Store.ListProducts()
	if err != nil {
		fmt.Printf("[HANDLER] GetAllProducts: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.JSON(w, http.StatusOK, products)
}

// handles GET /api/products/{id}.
func GetProduct(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	logFields := map[string]any{
		"handler": "GetProduct",
		"request": map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
		},
	}

	product, err := db.Store.GetProduct(id)
	
	if errors.Is(err, db.ErrNotFound) {
		logFields["error"] = map[string]any{
			"error type": "product not found",
			"status":  http.StatusNotFound,
			"id":      id,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logFields["status"] = http.StatusNotFound
		logger.ErrorContext(r.Context(), "product not found", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] GetProduct: product id=%s not found\n", id)
		utils.Error(w, http.StatusNotFound, "product not found")
		return
	}

	if err != nil {
		logFields["error"] = map[string]any{
			"error type": "db error",
			"status":  http.StatusInternalServerError,
			"id":      id,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logFields["status"] = http.StatusInternalServerError
		logger.ErrorContext(r.Context(), "db error", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] GetProduct: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logFields["status"] = http.StatusOK
	
	logFields["product"] = map[string]any{
		"id":          product.ID,
		"name":        product.Name,
		"description": product.Description,
		"price":       product.Price,
		"stock":       product.Stock,
	}
	
	logger.InfoContext(r.Context(), "product retrieved successfully", utils.MapToSlogAttrs(logFields)...)
	fmt.Printf("[HANDLER] GetProduct: product id=%s retrieved successfully\n", id)

	utils.JSON(w, http.StatusOK, product)
}

// handles POST /api/products.
func CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest

	logFields := map[string]any{
		"handler": "CreateProduct",
		"request": map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
		},
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logFields["error"] = map[string]any{
			"error type": "invalid JSON body",
			"status":  http.StatusBadRequest,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "invalid JSON body", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] CreateProduct: invalid JSON body: %v\n", err)
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)

	if req.Name == "" {
		logFields["error"] = map[string]any{
			"error type": "missing required field",
			"status":  http.StatusBadRequest,
			"field": "name",
			"error message": "name is required",
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "missing required field", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] CreateProduct: missing required field name\n")
		utils.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Price < 0 {
		logFields["error"] = map[string]any{
			"error type": "invalid field value",
			"status":  http.StatusBadRequest,
			"field": "price",
			"error message": "price must be non-negative",
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "invalid field value", utils.MapToSlogAttrs(logFields)...)

		fmt.Printf("[HANDLER] CreateProduct: negative price %.2f\n", req.Price)
		utils.Error(w, http.StatusBadRequest, "price must be non-negative")
		return
	}
	if req.Stock < 0 {
		logFields["error"] = map[string]any{
			"error type": "invalid field value",
			"status":  http.StatusBadRequest,
			"field": "stock",
			"error message": "stock must be non-negative",
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "invalid field value", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] CreateProduct: negative stock %d\n", req.Stock)
		utils.Error(w, http.StatusBadRequest, "stock must be non-negative")
		return
	}

	product := models.Product{
		ID:          utils.NewID(),
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		CreatedAt:   time.Now(),
	}

	logFields["product"] = map[string]any{
		"id":          product.ID,
		"name":        product.Name,
		"description": product.Description,
		"price":       product.Price,
		"stock":       product.Stock,
	}

	created, err := db.Store.CreateProduct(product)
	if err != nil {
		logFields["error"] = map[string]any{
			"error type": "database error",
			"status":  http.StatusInternalServerError,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "db error", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] CreateProduct: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.JSON(w, http.StatusCreated, created)
}

// handles PUT /api/products/{id}.
// Applies partial updates — only fields present in the body are changed.
func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	logFields := map[string]any{
		"handler": "UpdateProduct",
		"request": map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
			"id":     id,
		},
	}

	existing, err := db.Store.GetProduct(id)
	if errors.Is(err, db.ErrNotFound) {
		logFields["error"] = map[string]any{
			"error type": "product not found",
			"status":  http.StatusNotFound,
			"id":      id,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "product not found", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] UpdateProduct: product id=%s not found\n", id)
		utils.Error(w, http.StatusNotFound, "product not found")
		return
	}
	
	if err != nil {
		logFields["error"] = map[string]any{
			"error type": "db error",
			"status":  http.StatusInternalServerError,
			"id":      id,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "db error", utils.MapToSlogAttrs(logFields)...)
	
		fmt.Printf("[HANDLER] UpdateProduct: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	var req updateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logFields["error"] = map[string]any{
			"error type": "invalid JSON body",
			"status":  http.StatusBadRequest,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "invalid JSON body", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] UpdateProduct: invalid JSON body: %v\n", err)
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			logFields["error"] = map[string]any{
				"error type": "validation error",
				"status":  http.StatusBadRequest,
				"field": "name",
				"error message": "name must not be empty",
				"timestamp": time.Now().Format(time.RFC3339),
			}
			logger.ErrorContext(r.Context(), "validation error", utils.MapToSlogAttrs(logFields)...)
			fmt.Printf("[HANDLER] UpdateProduct: name must not be empty\n")
			utils.Error(w, http.StatusBadRequest, "name must not be empty")
			return
		}
		existing.Name = trimmed
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Price != nil {
		if *req.Price < 0 {
			logFields["error"] = map[string]any{
				"error type": "validation error",
				"status":  http.StatusBadRequest,
				"field": "price",
				"error message": "price must be non-negative",
				"timestamp": time.Now().Format(time.RFC3339),
			}
			logger.ErrorContext(r.Context(), "validation error", utils.MapToSlogAttrs(logFields)...)
			fmt.Printf("[HANDLER] UpdateProduct: price must be non-negative\n")
			utils.Error(w, http.StatusBadRequest, "price must be non-negative")
			return
		}
		existing.Price = *req.Price
	}
	if req.Stock != nil {
		if *req.Stock < 0 {
			logFields["error"] = map[string]any{
				"error type": "validation error",
				"status":  http.StatusBadRequest,
				"field": "stock",
				"error message": "stock must be non-negative",
				"timestamp": time.Now().Format(time.RFC3339),
			}
			logger.ErrorContext(r.Context(), "validation error", utils.MapToSlogAttrs(logFields)...)
			fmt.Printf("[HANDLER] UpdateProduct: stock must be non-negative\n")
			utils.Error(w, http.StatusBadRequest, "stock must be non-negative")
			return
		}
		existing.Stock = *req.Stock
	}

	if err := db.Store.UpdateProduct(existing); err != nil {
		logFields["error"] = map[string]any{
			"error type": "db error",
			"status":  http.StatusInternalServerError,
			"id":      id,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "db error", utils.MapToSlogAttrs(logFields)...)

		fmt.Printf("[HANDLER] UpdateProduct: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logFields["status"] = http.StatusOK
	logFields["updated_product"] = map[string]any{
		"id":          existing.ID,
		"name":        existing.Name,
		"description": existing.Description,
		"price":       existing.Price,
		"stock":       existing.Stock,
	}

	logger.InfoContext(r.Context(), "product updated successfully", utils.MapToSlogAttrs(logFields)...)
	fmt.Printf("[HANDLER] UpdateProduct: product id=%s updated successfully\n", id)
	utils.JSON(w, http.StatusOK, existing)
}

// handles DELETE /api/products/{id}.
func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	logFields := map[string]any{
		"handler": "DeleteProduct",
		"request": map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
			"id":     id,
		},
	}

	err := db.Store.DeleteProduct(id)
	if errors.Is(err, db.ErrNotFound) {
		logFields["error"] = map[string]any{
			"error type": "product not found",
			"status":  http.StatusNotFound,
			"id":      id,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "product not found", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] DeleteProduct: product id=%s not found\n", id)
		utils.Error(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		logFields["error"] = map[string]any{
			"error type": "db error",
			"status":  http.StatusInternalServerError,
			"id":      id,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "db error", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] DeleteProduct: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logFields["status"] = http.StatusNoContent
	logger.InfoContext(r.Context(), "product deleted successfully", utils.MapToSlogAttrs(logFields)...)
	fmt.Printf("[HANDLER] DeleteProduct: product id=%s deleted successfully\n", id)
	w.WriteHeader(http.StatusNoContent)
}
