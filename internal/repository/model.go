package repository

type FindClientByUsernameRequest struct {
	ClientID int
}

type FindClientByUsernameResponse struct {
	ClientID int    `json:"id" db:"id"`
	Username string `json:"username" db:"username"`
	Role     string `json:"role" db:"role"`
}

type SearchProductByNameRequest struct {
	ProductName string
}

type Product struct {
	ProductID          int32  `json:"id" db:"id"`
	ProductName        string `json:"name" db:"name"`
	ProductDescription string `json:"description" db:"description"`
	ProductPrice       string `json:"price" db:"price"`
}

type SearchProductByNameResponse struct {
	Products []Product
}

type CreateCartIfNotExistsRequest struct {
	ClientId int32 `json:"client_id" db:"client_id"`
}

type CreateCartIfNotExistsResponse struct {
	CartId int32 `json:"cart_id" db:"cart_id"`
}

type AddItemToCartRequest struct {
	ClientId  int32   `json:"client_id" db:"client_id"`
	CartId    int32   `json:"cart_id" db:"cart_id"`
	ProductID int32   `json:"product_id" db:"product_id"`
	Quantity  int32   `json:"quantity" db:"quantity"`
	Price     float64 `json:"price" db:"price"`
}

type AddItemToCartResponse struct {
	Success bool `json:"add success"`
}

type DeleteItemFromCartRequest struct {
	ClientId  int32 `json:"client_id" db:"client_id"`
	ProductID int32 `json:"product_id" db:"product_id"`
	CartId    int32 `json:"cart_id" db:"cart_id"`
}

type DeleteItemFromCartResponse struct {
	Success bool `json:"delete success"`
}

type GetCartRequest struct {
	ClientId int32 `json:"client_id" db:"client_id"`
	CartId   int32 `json:"cart_id" db:"cart_id"`
}

type CartItem struct {
	ProductID       int32   `json:"id" db:"id"`
	ProductQuantity int32   `json:"quantity" db:"quantity"`
	ProductPrice    float64 `json:"price" db:"price"`
}

type GetCartResponse struct {
	CartItems  []CartItem
	TotalPrice string
}

type PaymentRequest struct {
	ClientId int32 `json:"client_id" db:"client_id"`
}

type PaymentResponse struct {
	Success bool `json:"payment success"`
	Message string
}

type SaveKafkaMessageRequest struct {
	KafkaKey     string
	KafkaMessage interface{}
}

type SaveKafkaMessageResponse struct {
	Success bool `json:"save success"`
}

type GetKafkaMessageResponse struct {
	ID      int
	Key     string
	Message string
}

type SearchProductEvent struct {
	ProductName string    `json:"product_name"`
	Products    []Product `json:"products"`
	Message     string    `json:"message"`
}

type AddEvent struct {
	ClientID  int    `json:"client_id"`
	ProductID int32  `json:"product_id" db:"product_id"`
	Quantity  int32  `json:"quantity" db:"quantity"`
	Message   string `json:"message"`
}

type DeleteEvent struct {
	ClientID int    `json:"client_id"`
	Message  string `json:"message"`
}

type GetCartEvent struct {
	ClientID   int        `json:"client_id"`
	CartItems  []CartItem `json:"cart_items"`
	TotalPrice string     `json:"total_price"`
	Message    string     `json:"message"`
}

type PaymentEvent struct {
	ClientID int    `json:"client_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}
