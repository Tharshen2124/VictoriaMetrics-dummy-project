package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"tharshen2124/vivmdummyproject/db"
	"tharshen2124/vivmdummyproject/models"
	"tharshen2124/vivmdummyproject/utils"
)

// createOrderItemRequest is one line item in a new order request.
type createOrderItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// createOrderRequest is the expected JSON body for POST /api/orders.
type createOrderRequest struct {
	UserID string                   `json:"user_id"`
	Items  []createOrderItemRequest `json:"items"`
}

// updateOrderStatusRequest is the expected JSON body for
// PUT /api/orders/{id}/status.
type updateOrderStatusRequest struct {
	Status models.OrderStatus `json:"status"`
}

// CreateOrder handles POST /api/orders.
//
// Business rules enforced:
//   - user and every referenced product must exist
//   - each item must have quantity >= 1
//   - available stock must cover the requested quantity for every item
//   - stock is deducted atomically (all-or-nothing) together with order creation
//   - total amount is computed from per-product prices at time of order
func CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("[HANDLER] CreateOrder: invalid JSON body: %v\n", err)
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.UserID == "" {
		fmt.Printf("[HANDLER] CreateOrder: missing user_id\n")
		utils.Error(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if len(req.Items) == 0 {
		fmt.Printf("[HANDLER] CreateOrder: no items provided\n")
		utils.Error(w, http.StatusBadRequest, "at least one item is required")
		return
	}

	// Verify the user exists.
	if _, err := db.Store.GetUser(req.UserID); errors.Is(err, db.ErrNotFound) {
		fmt.Printf("[HANDLER] CreateOrder: user id=%s not found\n", req.UserID)
		utils.Error(w, http.StatusNotFound, "user not found")
		return
	} else if err != nil {
		fmt.Printf("[HANDLER] CreateOrder: GetUser db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Generate the order ID upfront so it can be referenced in order items.
	orderID := utils.NewID()

	// Resolve unit prices and build OrderItems.
	orderItems := make([]models.OrderItem, 0, len(req.Items))
	var total float64

	for _, item := range req.Items {
		if item.ProductID == "" {
			fmt.Printf("[HANDLER] CreateOrder: item missing product_id\n")
			utils.Error(w, http.StatusBadRequest, "each item must have a product_id")
			return
		}
		if item.Quantity < 1 {
			fmt.Printf("[HANDLER] CreateOrder: item product_id=%s has quantity %d\n", item.ProductID, item.Quantity)
			utils.Error(w, http.StatusBadRequest, "item quantity must be at least 1")
			return
		}

		product, err := db.Store.GetProduct(item.ProductID)
		if errors.Is(err, db.ErrNotFound) {
			fmt.Printf("[HANDLER] CreateOrder: product id=%s not found\n", item.ProductID)
			utils.Error(w, http.StatusNotFound, fmt.Sprintf("product %s not found", item.ProductID))
			return
		}
		if err != nil {
			fmt.Printf("[HANDLER] CreateOrder: GetProduct db error: %v\n", err)
			utils.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}

		orderItems = append(orderItems, models.OrderItem{
			ID:        utils.NewID(),
			OrderID:   orderID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: product.Price,
		})
		total += product.Price * float64(item.Quantity)
	}

	order := models.Order{
		ID:          orderID,
		UserID:      req.UserID,
		Items:       orderItems,
		TotalAmount: total,
		Status:      models.OrderStatusPending,
		CreatedAt:   time.Now(),
	}

	// CreateOrder handles stock deduction and order persistence atomically.
	created, err := db.Store.CreateOrder(order)
	if err != nil {
		var conflictErr *db.ConflictError
		if errors.As(err, &conflictErr) {
			fmt.Printf("[HANDLER] CreateOrder: conflict: %s\n", conflictErr.Reason)
			utils.Error(w, http.StatusConflict, conflictErr.Reason)
			return
		}
		fmt.Printf("[HANDLER] CreateOrder: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.JSON(w, http.StatusCreated, created)
}

// GetOrder handles GET /api/orders/{id}.
func GetOrder(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	order, err := db.Store.GetOrder(id)
	if errors.Is(err, db.ErrNotFound) {
		fmt.Printf("[HANDLER] GetOrder: order id=%s not found\n", id)
		utils.Error(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		fmt.Printf("[HANDLER] GetOrder: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.JSON(w, http.StatusOK, order)
}

// ListOrdersByUser handles GET /api/users/{id}/orders.
func ListOrdersByUser(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]

	// Confirm the user exists before attempting the order look-up.
	if _, err := db.Store.GetUser(userID); errors.Is(err, db.ErrNotFound) {
		fmt.Printf("[HANDLER] ListOrdersByUser: user id=%s not found\n", userID)
		utils.Error(w, http.StatusNotFound, "user not found")
		return
	} else if err != nil {
		fmt.Printf("[HANDLER] ListOrdersByUser: GetUser db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	orders, err := db.Store.ListOrdersByUser(userID)
	if err != nil {
		fmt.Printf("[HANDLER] ListOrdersByUser: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.JSON(w, http.StatusOK, orders)
}

// UpdateOrderStatus handles PUT /api/orders/{id}/status.
// Allowed target statuses: pending, completed, cancelled.
func UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req updateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("[HANDLER] UpdateOrderStatus: invalid JSON body: %v\n", err)
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	switch req.Status {
	case models.OrderStatusPending, models.OrderStatusCompleted, models.OrderStatusCancelled:
		// valid
	default:
		fmt.Printf("[HANDLER] UpdateOrderStatus: invalid status %q\n", req.Status)
		utils.Error(w, http.StatusBadRequest, "status must be one of: pending, completed, cancelled")
		return
	}

	if err := db.Store.UpdateOrderStatus(id, req.Status); errors.Is(err, db.ErrNotFound) {
		fmt.Printf("[HANDLER] UpdateOrderStatus: order id=%s not found\n", id)
		utils.Error(w, http.StatusNotFound, "order not found")
		return
	} else if err != nil {
		fmt.Printf("[HANDLER] UpdateOrderStatus: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	order, err := db.Store.GetOrder(id)
	if err != nil {
		fmt.Printf("[HANDLER] UpdateOrderStatus: GetOrder db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.JSON(w, http.StatusOK, order)
}
