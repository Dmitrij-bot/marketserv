package usecase

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
	ClientId int `json:"client_id" db:"client_id"`
}

type CreateCartIfNotExistsResponse struct {
	CartId int `json:"cart_id" db:"cart_id"`
}

type AddItemToCartRequest struct {
	CartId    int   `json:"cart_id" db:"cart_id"`
	ProductID int32 `json:"product_id" db:"product_id"`
	Quantity  int   `json:"quantity" db:"quantity"`
}

type AddItemToCartResponse struct {
	Success bool `json:"add success"`
}
