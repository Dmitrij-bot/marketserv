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
    WITH updated AS (
        UPDATE products
        SET quantity = quantity - $3
        WHERE id = $2 AND quantity >= $3
        RETURNING id
    )
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
    WHERE EXISTS (SELECT 1 FROM updated);
`
	SearchProductByIdSQL   = "SELECT EXISTS(SELECT 1 FROM cart_items WHERE cart_id = $1 AND product_id = $2)"
	DeleteItemFromCartSQL  = "DELETE FROM cart_items  WHERE  cart_id = $1 AND  product_id = $2"
	DeleteItemFromCartSQL2 = `
    WITH updated AS (
        UPDATE cart_items
        SET quantity = quantity - 1
        WHERE cart_id = $1 AND  product_id = $2 AND quantity >0
        RETURNING quantity
    ),
     deleted AS (
        DELETE FROM cart_items
        WHERE cart_id = $1 AND  product_id = $2 AND quantity = 0
    )
    UPDATE products
        SET quantity = quantity + 1
        WHERE id = $2
        AND EXISTS (SELECT 1 FROM updated);
`
	GetCartItemSQL = "SELECT product_id, quantity, price FROM cart_items WHERE cart_id = $1"
	PaymentSQL     = `
    WITH total AS (
        SELECT SUM(ci.quantity * ci.price) AS total_price
        FROM cart_items ci
        JOIN carts c ON c.cart_id = ci.cart_id
        WHERE c.user_id = $1
    ),
    updated AS (
        UPDATE clients_table
        SET invoice = invoice - (SELECT total_price FROM total)
        WHERE id = $1 AND invoice >= (SELECT total_price FROM total)
        RETURNING id
    ),
    wallet_update AS (
        UPDATE wallet_market
        SET balance = balance + (SELECT total_price FROM total)
        WHERE id = 1 
    )
    DELETE FROM cart_items
    WHERE cart_id = (SELECT cart_id FROM carts WHERE user_id = $1)
    AND EXISTS (SELECT 1 FROM updated);
    `
)
