package usecase

type FindClientByUsernameRequest struct {
	ClientID int
}

type FindClientByUsernameResponse struct {
	ClientID int    `json:"id" db:"id"`
	Username string `json:"username" db:"username"`
	Role     string `json:"role" db:"role"`
}
