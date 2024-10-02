package repository

const (
	FindClientByUserNameSql = "SELECT id,username,role FROM clients_table WHERE id = $1"
)
