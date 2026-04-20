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
	logFields := map[string]any{
		"handler": "CreateOrder",
		"request": map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
		},
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logFields["error"] = map[string]any{
			"error type": "validation error",
			"status":  http.StatusBadRequest,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "validation error", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] CreateOrder: invalid JSON body: %v\n", err)
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.UserID == "" {
		logFields["error"] = map[string]any{
			"error type": "validation error",
			"status":  http.StatusBadRequest,
			"error message": "user_id is required",
			"stack trace": fmt.Sprintf("%+v", req),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "validation error", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] CreateOrder: missing user_id\n")
		utils.Error(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if len(req.Items) == 0 {
		logFields["error"] = map[string]any{
			"error type": "validation error",
			"status":  http.StatusBadRequest,
			"error message": "at least one item is required",
			"stack trace": fmt.Sprintf("%+v", req),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "validation error", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] CreateOrder: no items provided\n")
		utils.Error(w, http.StatusBadRequest, "at least one item is required")
		return
	}

	// Verify the user exists.
	if _, err := db.Store.GetUser(req.UserID); errors.Is(err, db.ErrNotFound) {
		logFields["error"] = map[string]any{
			"error type": "validation error",
			"status":  http.StatusNotFound,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", req),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "validation error", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] CreateOrder: user id=%s not found\n", req.UserID)
		utils.Error(w, http.StatusNotFound, "user not found")
		return
	} else if err != nil {
		logFields["error"] = map[string]any{
			"error type": "db error",
			"status":  http.StatusInternalServerError,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", req),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "db error", utils.MapToSlogAttrs(logFields)...)
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
			logFields["error"] = map[string]any{
				"error type": "validation error",
				"status":  http.StatusBadRequest,
				"error message": "each item must have a product_id",
				"stack trace": fmt.Sprintf("%+v", item),
				"timestamp": time.Now().Format(time.RFC3339),
			}
			logger.ErrorContext(r.Context(), "validation error", utils.MapToSlogAttrs(logFields)...)
			fmt.Printf("[HANDLER] CreateOrder: item missing product_id\n")
			utils.Error(w, http.StatusBadRequest, "each item must have a product_id")
			return
		}
		if item.Quantity < 1 {
			logFields["error"] = map[string]any{
				"error type": "validation error",
				"status":  http.StatusBadRequest,
				"error message": "item quantity must be at least 1",
				"stack trace": fmt.Sprintf("%+v", item),
				"timestamp": time.Now().Format(time.RFC3339),
			}
			logger.ErrorContext(r.Context(), "validation error", utils.MapToSlogAttrs(logFields)...)					
			fmt.Printf("[HANDLER] CreateOrder: item product_id=%s has quantity %d\n", item.ProductID, item.Quantity)
			utils.Error(w, http.StatusBadRequest, "item quantity must be at least 1")
			return
		}

		product, err := db.Store.GetProduct(item.ProductID)
		if errors.Is(err, db.ErrNotFound) {
			logFields["error"] = map[string]any{
				"error type": "validation error",
				"status":  http.StatusNotFound,
				"error message": fmt.Sprintf("product %s not found", item.ProductID),
				"stack trace": fmt.Sprintf("%+v", item),
				"timestamp": time.Now().Format(time.RFC3339),
			}
			logger.ErrorContext(r.Context(), "validation error", utils.MapToSlogAttrs(logFields)...)
			fmt.Printf("[HANDLER] CreateOrder: product id=%s not found\n", item.ProductID)
			utils.Error(w, http.StatusNotFound, fmt.Sprintf("product %s not found", item.ProductID))
			return
		}
		if err != nil {
			logFields["error"] = map[string]any{
				"error type": "db error",
				"status":  http.StatusInternalServerError,
				"error message": err.Error(),
				"stack trace": fmt.Sprintf("%+v", item),
				"timestamp": time.Now().Format(time.RFC3339),
			}
			logger.ErrorContext(r.Context(), "db error", utils.MapToSlogAttrs(logFields)...)
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

	logFields["order"] = map[string]any{
		"id": order.ID,
		"user_id": order.UserID,
		"total_amount": order.TotalAmount,
		"item_count": len(order.Items),
		"items": func() []map[string]any {
			items := make([]map[string]any, 0, len(order.Items))
			for _, item := range order.Items {
				items = append(items, map[string]any{
					"product_id": item.ProductID,
					"quantity": item.Quantity,
					"unit_price": item.UnitPrice,
				})
			}
			return items
		}(),
		"timestamp": time.Now().Format(time.RFC3339),
		"status": order.Status,
	}

	// CreateOrder handles stock deduction and order persistence atomically.
	created, err := db.Store.CreateOrder(order)
	if err != nil {
		var conflictErr *db.ConflictError
		if errors.As(err, &conflictErr) {
			logFields["error"] = map[string]any{
				"error type": "conflict error",
				"status":  http.StatusConflict,
				"error message": conflictErr.Reason,
				"stack trace": fmt.Sprintf("%+v", conflictErr),
				"timestamp": time.Now().Format(time.RFC3339),
			}
			logger.ErrorContext(r.Context(), "conflict error", utils.MapToSlogAttrs(logFields)...)
			
			fmt.Printf("[HANDLER] CreateOrder: conflict: %s\n", conflictErr.Reason)
			utils.Error(w, http.StatusConflict, conflictErr.Reason)
			return
		}
		logFields["error"] = map[string]any{
			"error type": "db error",
			"status":  http.StatusInternalServerError,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "db error", utils.MapToSlogAttrs(logFields)...)
		
		fmt.Printf("[HANDLER] CreateOrder: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.JSON(w, http.StatusCreated, created)
}

// GetOrder handles GET /api/orders/{id}.
func GetOrder(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	logFields := map[string]any{
		"handler": "GetOrder",
		"method":  r.Method,
		"path":    r.URL.Path,
		"order_id_by_path": id,
	}

	order, err := db.Store.GetOrder(id)
	if errors.Is(err, db.ErrNotFound) {
		logFields["error"] = map[string]any{
			"error type": "not found",
			"status":  http.StatusNotFound,
			"error message": "order not found",
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "order not found", utils.MapToSlogAttrs(logFields)...)
		
		fmt.Printf("[HANDLER] GetOrder: order id=%s not found\n", id)
		utils.Error(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		logFields["error"] = map[string]any{
			"error type": "db error",
			"status":  http.StatusInternalServerError,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "db error", utils.MapToSlogAttrs(logFields)...)

		fmt.Printf("[HANDLER] GetOrder: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logFields["order"] = map[string]any{
		"order_id": order.ID,
		"user_id": order.UserID,
		"total_amount": order.TotalAmount,
		"item_count": len(order.Items),
		"items": func() []map[string]any {
			items := make([]map[string]any, 0, len(order.Items))
			for _, item := range order.Items {
				items = append(items, map[string]any{
					"product_id": item.ProductID,
					"quantity": item.Quantity,
					"unit_price": item.UnitPrice,
				})
			}
			return items
		}(),
		"timestamp": time.Now().Format(time.RFC3339),
		"status": order.Status,
	}

	logFields["status"] = http.StatusOK
	logger.InfoContext(r.Context(), "order retrieved successfully", utils.MapToSlogAttrs(logFields)...)

	utils.JSON(w, http.StatusOK, order)
}

// ListOrdersByUser handles GET /api/users/{id}/orders.
func ListOrdersByUser(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]

	logFields := map[string]any{
		"handler": "ListOrdersByUser",
		"method":  r.Method,
		"path":    r.URL.Path,
		"user_id": userID,
	}

	// Confirm the user exists before attempting the order look-up.
	if _, err := db.Store.GetUser(userID); errors.Is(err, db.ErrNotFound) {
		logFields["error"] = map[string]any{
			"error type": "not found",
			"status":  http.StatusNotFound,
			"error message": "user not found",
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "user not found", utils.MapToSlogAttrs(logFields)...)
		
		fmt.Printf("[HANDLER] ListOrdersByUser: user id=%s not found\n", userID)
		utils.Error(w, http.StatusNotFound, "user not found")
		return
	} else if err != nil {
		logFields["error"] = map[string]any{
			"error type": "db error",
			"status":  http.StatusInternalServerError,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "db error", utils.MapToSlogAttrs(logFields)...)
		
		fmt.Printf("[HANDLER] ListOrdersByUser: GetUser db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	orders, err := db.Store.ListOrdersByUser(userID)
	if err != nil {
		logFields["error"] = map[string]any{
			"error type": "db error",
			"status":  http.StatusInternalServerError,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "db error", utils.MapToSlogAttrs(logFields)...)
		
		fmt.Printf("[HANDLER] ListOrdersByUser: ListOrdersByUser db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	logFields["order_count"] = len(orders)
	logFields["status"] = http.StatusOK
	logger.InfoContext(r.Context(), "orders listed successfully", utils.MapToSlogAttrs(logFields)...)

	utils.JSON(w, http.StatusOK, orders)
}

// UpdateOrderStatus handles PUT /api/orders/{id}/status.
// Allowed target statuses: pending, completed, cancelled.
func UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	logFields := map[string]any{
		"handler": "UpdateOrderStatus",
		"method":  r.Method,
		"path":    r.URL.Path,
		"order_id": id,
	}

	var req updateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logFields["error"] = map[string]any{
			"error type": "invalid JSON",
			"status":  http.StatusBadRequest,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "invalid JSON body", utils.MapToSlogAttrs(logFields)...)
		
		fmt.Printf("[HANDLER] UpdateOrderStatus: invalid JSON body: %v\n", err)
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	switch req.Status {
	case models.OrderStatusPending, models.OrderStatusCompleted, models.OrderStatusCancelled:
		// valid
	default:
		logFields["error"] = map[string]any{
			"error type": "invalid status",
			"status":  http.StatusBadRequest,
			"error message": fmt.Sprintf("status must be one of: pending, completed, cancelled"),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "invalid status", utils.MapToSlogAttrs(logFields)...)
		
		fmt.Printf("[HANDLER] UpdateOrderStatus: invalid status %q\n", req.Status)
		utils.Error(w, http.StatusBadRequest, "status must be one of: pending, completed, cancelled")
		return
	}

	if err := db.Store.UpdateOrderStatus(id, req.Status); errors.Is(err, db.ErrNotFound) {
		logFields["error"] = map[string]any{
			"error type": "order not found",
			"status":  http.StatusNotFound,
			"stack trace": fmt.Sprintf("%+v", err),
			"error message": fmt.Sprintf("order id=%s not found", id),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "order not found", utils.MapToSlogAttrs(logFields)...)
		
		fmt.Printf("[HANDLER] UpdateOrderStatus: order id=%s not found\n", id)
		utils.Error(w, http.StatusNotFound, "order not found")
		return
	} else if err != nil {
		logFields["error"] = map[string]any{
			"error type": "db error",
			"status":  http.StatusInternalServerError,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "db error", utils.MapToSlogAttrs(logFields)...)
		
		fmt.Printf("[HANDLER] UpdateOrderStatus: db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	order, err := db.Store.GetOrder(id)
	if err != nil {
		logFields["error"] = map[string]any{
			"error type": "db error",
			"status":  http.StatusInternalServerError,
			"error message": err.Error(),
			"stack trace": fmt.Sprintf("%+v", err),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		logger.ErrorContext(r.Context(), "db error", utils.MapToSlogAttrs(logFields)...)
		fmt.Printf("[HANDLER] UpdateOrderStatus: GetOrder db error: %v\n", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	logFields["order"] = map[string]any{
		"id": order.ID,
		"user_id": order.UserID,
		"total_amount": order.TotalAmount,
		"item_count": len(order.Items),
		"items": func() []map[string]any {
			items := make([]map[string]any, 0, len(order.Items))
			for _, item := range order.Items {
				items = append(items, map[string]any{
					"product_id": item.ProductID,
					"quantity": item.Quantity,
					"unit_price": item.UnitPrice,
				})
			}
			return items
		}(),
		"timestamp": time.Now().Format(time.RFC3339),
		"status": order.Status,
	}
	
	logFields["status"] = http.StatusOK
	logger.InfoContext(r.Context(), "order status updated successfully", utils.MapToSlogAttrs(logFields)...)

	utils.JSON(w, http.StatusOK, order)
}
