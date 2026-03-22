package routes

import (
	"github.com/gorilla/mux"

	"tharshen2124/vivmdummyproject/handlers"
	"tharshen2124/vivmdummyproject/middlewares"
)

// can call from tests with a fresh router.
func RegisterRoutes(router *mux.Router) {
	// Apply the logging middleware to every route registered on this router.
	router.Use(middlewares.Logging)

	api := router.PathPrefix("/api").Subrouter()

	api.HandleFunc("/users", handlers.CreateUser).Methods("POST")
	api.HandleFunc("/users/{id}", handlers.GetUser).Methods("GET")
	api.HandleFunc("/users/{id}/orders", handlers.ListOrdersByUser).Methods("GET")

	api.HandleFunc("/products", handlers.GetAllProducts).Methods("GET")
	api.HandleFunc("/products", handlers.CreateProduct).Methods("POST")
	api.HandleFunc("/products/{id}", handlers.GetProduct).Methods("GET")
	api.HandleFunc("/products/{id}", handlers.UpdateProduct).Methods("PUT")
	api.HandleFunc("/products/{id}", handlers.DeleteProduct).Methods("DELETE")

	api.HandleFunc("/orders", handlers.CreateOrder).Methods("POST")
	api.HandleFunc("/orders/{id}", handlers.GetOrder).Methods("GET")
	api.HandleFunc("/orders/{id}/status", handlers.UpdateOrderStatus).Methods("PUT")
}
