package repository

const (
	FindClientByUserNameSql = "SELECT id,username,role FROM clients_table WHERE id = $1"
	SearchProductByNameSQL  = "SELECT id, name, description, price FROM products WHERE name ILIKE '%' || $1 || '%'"
	GetCartSQL              = "SELECT cart_id FROM carts WHERE user_id = $1"

	CreateCartIfNotExistsSQL = `
            INSERT INTO carts (user_id, created_at, updated_at) 
            VALUES ($1, NOW(), NOW()) 
            ON CONFLICT (user_id) DO NOTHING 
            RETURNING cart_id`

	AddItemToCartSQL = `
    INSERT INTO cart_items (cart_id, product_id, quantity, price, added_at)
    VALUES (
        $1, -- cart_id
        $2, -- product_id
        $3, -- quantity
        (SELECT price FROM products WHERE id = $2 LIMIT 1), -- price
        NOW() -- added_at
    )
    ON CONFLICT (cart_id, product_id)
    DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity
`
	SearchProductByIdSQL  = "SELECT EXISTS(SELECT 1 FROM cart_items WHERE cart_id = $1 AND product_id = $2)"
	DeleteItemFromCartSQL = "DELETE FROM cart_items  WHERE  cart_id = $1 AND  product_id = $2"
	GetCartItemSQL        = "SELECT product_id, quantity, price FROM cart_items WHERE cart_id = $1"
)
