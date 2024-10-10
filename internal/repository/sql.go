package repository

const (
	FindClientByUserNameSql = "SELECT id,username,role FROM clients_table WHERE id = $1"
	SearchProductByNameSQL  = "SELECT id,name,description,price FROM products WHERE name =1$"
)
