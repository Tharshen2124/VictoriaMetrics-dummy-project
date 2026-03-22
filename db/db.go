package db

import (
	"errors"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"tharshen2124/vivmdummyproject/models"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// ConflictError is returned when a business rule is violated (e.g. insufficient stock).
type ConflictError struct {
	Reason string
}

func (e *ConflictError) Error() string { return e.Reason }

// PostgresStore wraps a gorm.DB connection and exposes all data-access methods.
type PostgresStore struct {
	db *gorm.DB
}

// Store is the package-level singleton accessed by all handlers.
var Store *PostgresStore

// InitDB opens a GORM connection to the Postgres database and stores it in Store.
func InitDB(dsn string) error {
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	Store = &PostgresStore{db: gormDB}
	fmt.Println("[DB] Connected to postgres")
	return nil
}

// --- User operations ---

func (s *PostgresStore) CreateUser(u models.User) (models.User, error) {
	return u, s.db.Create(&u).Error
}

func (s *PostgresStore) GetUser(id string) (models.User, error) {
	var u models.User
	err := s.db.First(&u, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *PostgresStore) EmailExists(email string) (bool, error) {
	var count int64
	err := s.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

// --- Product operations ---

func (s *PostgresStore) CreateProduct(p models.Product) (models.Product, error) {
	return p, s.db.Create(&p).Error
}

func (s *PostgresStore) GetProduct(id string) (models.Product, error) {
	var p models.Product
	err := s.db.First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return p, ErrNotFound
	}
	return p, err
}

func (s *PostgresStore) ListProducts() ([]models.Product, error) {
	var products []models.Product
	err := s.db.Find(&products).Error
	if products == nil {
		products = []models.Product{}
	}
	return products, err
}

func (s *PostgresStore) UpdateProduct(p models.Product) error {
	result := s.db.Model(&p).Updates(map[string]any{
		"name":        p.Name,
		"description": p.Description,
		"price":       p.Price,
		"stock":       p.Stock,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteProduct(id string) error {
	result := s.db.Delete(&models.Product{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Order operations ---

// CreateOrder persists the order and its items, and deducts product stock — all
// within a single transaction so the operation is atomic.
func (s *PostgresStore) CreateOrder(o models.Order) (models.Order, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range o.Items {
			result := tx.Model(&models.Product{}).
				Where("id = ? AND stock >= ?", item.ProductID, item.Quantity).
				UpdateColumn("stock", gorm.Expr("stock - ?", item.Quantity))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				var product models.Product
				if err := tx.First(&product, "id = ?", item.ProductID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
					return &ConflictError{Reason: fmt.Sprintf("product %s not found", item.ProductID)}
				}
				return &ConflictError{Reason: fmt.Sprintf("insufficient stock for product %s (have %d, want %d)", item.ProductID, product.Stock, item.Quantity)}
			}
		}
		return tx.Create(&o).Error
	})
	return o, err
}

func (s *PostgresStore) GetOrder(id string) (models.Order, error) {
	var o models.Order
	err := s.db.Preload("Items").First(&o, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return o, ErrNotFound
	}
	return o, err
}

func (s *PostgresStore) ListOrdersByUser(userID string) ([]models.Order, error) {
	var orders []models.Order
	err := s.db.Preload("Items").Where("user_id = ?", userID).Find(&orders).Error
	if orders == nil {
		orders = []models.Order{}
	}
	return orders, err
}

func (s *PostgresStore) UpdateOrderStatus(id string, status models.OrderStatus) error {
	result := s.db.Model(&models.Order{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
