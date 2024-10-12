package repository

const (
	FindClientByUserNameSql        = "SELECT id,username,role FROM clients_table WHERE id = $1"
	SearchProductByNameSQL         = "SELECT id, name, description, price FROM products WHERE name ILIKE '%' || $1 || '%'"
	CreateCartIfNotExistsInsertSQL = `
    INSERT INTO carts (user_id) 
    VALUES ($1)
    ON CONFLICT (user_id) DO NOTHING;
`
	CreateCartIfNotExistsSelectSQL = `
    SELECT cart_id 
    FROM carts 
    WHERE user_id = $1
    LIMIT 1;
`

	AddItemToCartSQL = `
        INSERT INTO cart_items (cart_id, product_id, quantity, price)
        VALUES ($1, $2, $3, (SELECT price FROM products WHERE id = $2 LIMIT 1))
        ON CONFLICT (cart_id, product_id)
        DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity
    `
)
