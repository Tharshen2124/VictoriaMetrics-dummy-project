# Architecture

## 1. Folder Structure

```
.
├── main.go                  Entry point — loads config, inits store, starts HTTP server
├── config/
│   └── config.go            Reads PORT and DATABASE_URL from .env via spf13/viper
├── db/
│   └── db.go                Thread-safe in-memory store (sync.RWMutex maps); seeds sample products
├── models/
│   ├── user.go              User domain type
│   ├── product.go           Product domain type
│   └── order.go             Order + OrderItem domain types; OrderStatus constants
├── handlers/
│   ├── user_handler.go      HTTP handlers for /api/users endpoints
│   ├── product_handler.go   HTTP handlers for /api/products endpoints
│   └── order_handler.go     HTTP handlers for /api/orders endpoints
├── routes/
│   └── routes.go            Registers all routes on a gorilla/mux router; attaches middleware
├── middlewares/
│   └── logging.go           Per-request logging: method, path, status code, duration
└── utils/
    └── response.go          JSON() and Error() response helpers; NewID() random hex ID generator
```

## 2. Request Lifecycle

```
HTTP Request
    │
    ▼
gorilla/mux Router
    │  (matches path + method)
    ▼
middlewares.Logging          ← wraps ResponseWriter to capture status code
    │  records start time
    ▼
Handler function             ← decodes body, validates input, calls db.Store methods
    │  e.g. handlers.CreateOrder
    ▼
db.InMemoryStore             ← acquires mutex, reads/writes maps, logs [DB] events
    │
    ▼
utils.JSON / utils.Error     ← sets Content-Type, writes status + JSON body
    │
    ▼
middlewares.Logging          ← prints [HTTP] line with method/path/status/duration
    │
    ▼
HTTP Response
```

## 3. Entities and Relationships

```
User  1 ──────────── * Order
                          │
                          │  contains 1..*
                          ▼
                      OrderItem  * ──── 1  Product
```

- A **User** can place many **Orders**.
- Each **Order** contains one or more **OrderItems**.
- Each **OrderItem** references exactly one **Product** and captures the `unit_price` at the moment the order was placed (so future price changes do not retroactively alter historical orders).
- When an order is created, stock is deducted from each referenced **Product** atomically (all-or-nothing via a single lock acquisition in `db.DeductStock`).

## 4. API Endpoint Table

| Method | Path                        | Purpose                                      |
|--------|-----------------------------|----------------------------------------------|
| POST   | /api/users                  | Register a new user                          |
| GET    | /api/users/{id}             | Get a user by ID                             |
| GET    | /api/users/{id}/orders      | List all orders belonging to a user          |
| GET    | /api/products               | List all products                            |
| POST   | /api/products               | Create a new product                         |
| GET    | /api/products/{id}          | Get a product by ID                          |
| PUT    | /api/products/{id}          | Partially update a product (price, stock, …) |
| DELETE | /api/products/{id}          | Delete a product                             |
| POST   | /api/orders                 | Create an order (validates stock, deducts)   |
| GET    | /api/orders/{id}            | Get an order by ID                           |
| PUT    | /api/orders/{id}/status     | Update order status (pending/completed/cancelled) |

### Request / Response shapes

**POST /api/users**
```json
// Request
{ "name": "Alice", "email": "alice@example.com" }

// 201 Created
{ "id": "...", "name": "Alice", "email": "alice@example.com", "created_at": "..." }
```

**POST /api/products**
```json
// Request
{ "name": "Widget", "description": "...", "price": 9.99, "stock": 100 }

// 201 Created — full Product object
```

**PUT /api/products/{id}**
```json
// Request — all fields optional, only provided fields are changed
{ "price": 12.99, "stock": 80 }

// 200 OK — updated Product object
```

**POST /api/orders**
```json
// Request
{
  "user_id": "<user-id>",
  "items": [
    { "product_id": "<product-id>", "quantity": 2 }
  ]
}

// 201 Created
{
  "id": "...",
  "user_id": "...",
  "items": [{ "product_id": "...", "quantity": 2, "unit_price": 9.99 }],
  "total_amount": 19.98,
  "status": "pending",
  "created_at": "..."
}
```

**PUT /api/orders/{id}/status**
```json
// Request
{ "status": "completed" }

// 200 OK — updated Order object
```

All error responses have the shape:
```json
{ "error": "human-readable message" }
```

## 5. Debugging Guide

| Symptom | Where to look |
|---------|---------------|
| Order total is wrong | `handlers/order_handler.go` — `CreateOrder`, the `total +=` accumulation loop |
| Stock not being deducted | `db/db.go` — `DeductStock`; check the `[DB]` log lines for each product |
| Product not found | `db/db.go` — `Products` map; run a `GET /api/products` to see what IDs exist |
| User not found | `db/db.go` — `Users` map; confirm the UUID used in the request matches a `POST /api/users` response |
| Duplicate email error | `db/db.go` — `EmailExists` linear scan; email comparison is case-sensitive |
| Request not reaching handler | `routes/routes.go` — verify the method and path prefix `/api` are registered correctly |
| Status code not appearing in logs | `middlewares/logging.go` — the `responseWriter` wrapper; check `WriteHeader` is called by the handler |
| Server does not start | `main.go` + `config/config.go` — ensure `.env` contains `PORT=<number>` |
| Middleware not executing | `routes/routes.go` — `r.Use(middlewares.Logging)` must be called before handler registration |
