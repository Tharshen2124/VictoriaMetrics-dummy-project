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

	product, err := db.Store.GetProduct(id)
	if errors.Is(err, db.ErrNotFound) {
		fmt.Printf("[HANDLER] GetProduct: product id=%s not found\n", id)
		utils.Error(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		fmt.Printf("[HANDLER] GetProduct: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.JSON(w, http.StatusOK, product)
}

// handles POST /api/products.
func CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("[HANDLER] CreateProduct: invalid JSON body: %v\n", err)
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)

	if req.Name == "" {
		fmt.Printf("[HANDLER] CreateProduct: missing required field name\n")
		utils.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Price < 0 {
		fmt.Printf("[HANDLER] CreateProduct: negative price %.2f\n", req.Price)
		utils.Error(w, http.StatusBadRequest, "price must be non-negative")
		return
	}
	if req.Stock < 0 {
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

	created, err := db.Store.CreateProduct(product)
	if err != nil {
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

	existing, err := db.Store.GetProduct(id)
	if errors.Is(err, db.ErrNotFound) {
		fmt.Printf("[HANDLER] UpdateProduct: product id=%s not found\n", id)
		utils.Error(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		fmt.Printf("[HANDLER] UpdateProduct: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	var req updateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("[HANDLER] UpdateProduct: invalid JSON body: %v\n", err)
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
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
			utils.Error(w, http.StatusBadRequest, "price must be non-negative")
			return
		}
		existing.Price = *req.Price
	}
	if req.Stock != nil {
		if *req.Stock < 0 {
			utils.Error(w, http.StatusBadRequest, "stock must be non-negative")
			return
		}
		existing.Stock = *req.Stock
	}

	if err := db.Store.UpdateProduct(existing); err != nil {
		fmt.Printf("[HANDLER] UpdateProduct: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.JSON(w, http.StatusOK, existing)
}

// handles DELETE /api/products/{id}.
func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	err := db.Store.DeleteProduct(id)
	if errors.Is(err, db.ErrNotFound) {
		fmt.Printf("[HANDLER] DeleteProduct: product id=%s not found\n", id)
		utils.Error(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		fmt.Printf("[HANDLER] DeleteProduct: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
