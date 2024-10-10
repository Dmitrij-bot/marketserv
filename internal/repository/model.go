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

type SearchProductByNameResponse struct {
	ProductID          int32  `json:"id" db:"id"`
	ProductName        string `json:"name" db:"name"`
	ProductDescription string `json:"description" db:"description"`
	ProductPrice       string `json:"price" db:"price"`
}
