package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Dmitrij-bot/marketserv/pkg/postgres"
	"github.com/Dmitrij-bot/marketserv/pkg/redis"
	redis2 "github.com/go-redis/redis/v8"
	"log"
)

type UserRepository struct {
	db          *postgres.DB
	redisClient *redis.RedisDB
}

func NewUserRepository(db *postgres.DB, redisClient *redis.RedisDB) *UserRepository {
	return &UserRepository{
		db:          db,
		redisClient: redisClient,
	}
}

func (r *UserRepository) FindClientByUsername(ctx context.Context, req FindClientByUsernameRequest) (resp FindClientByUsernameResponse, err error) {
	err = r.db.QueryRowContext(ctx, FindClientByUserNameSql, req.ClientID).Scan(&resp.ClientID, &resp.Username, &resp.Role)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (r *UserRepository) SearchProductByName(ctx context.Context, req SearchProductByNameRequest) (resp SearchProductByNameResponse, err error) {

	if req.ProductName == "" {
		return SearchProductByNameResponse{}, errors.New("product name cannot be empty")
	}

	rows, err := r.db.QueryContext(ctx, SearchProductByNameSQL, req.ProductName)
	if err != nil {
		return SearchProductByNameResponse{}, fmt.Errorf("failed to query products: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close rows: %w", closeErr)
		}
	}()

	resp.Products = []Product{}

	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ProductID, &product.ProductName, &product.ProductDescription, &product.ProductPrice); err != nil {
			return resp, fmt.Errorf("failed to scan product: %w", err)
		}

		resp.Products = append(resp.Products, product)
	}

	if err := rows.Err(); err != nil {
		return resp, fmt.Errorf("error occurred during row iteration: %w", err)
	}

	return resp, nil
}

func (r *UserRepository) CreateCartIfNotExists(ctx context.Context, req CreateCartIfNotExistsRequest) (resp CreateCartIfNotExistsResponse, err error) {

	err = r.db.QueryRowContext(ctx, GetCartSQL, req.ClientId).Scan(&resp.CartId)

	if err == sql.ErrNoRows {
		err := r.db.QueryRowContext(ctx, CreateCartIfNotExistsSQL, req.ClientId).Scan(&resp.CartId)

		if err != nil {
			return resp, fmt.Errorf("failed to create cart: %w", err)
		}
	} else if err != nil {
		return resp, fmt.Errorf("failed to retrieve cart: %w", err)
	}

	return resp, nil
}

func (r *UserRepository) AddItemToCart(ctx context.Context, req AddItemToCartRequest) (resp AddItemToCartResponse, err error) {
	log.Printf("Received request: ProductID=%d, Quantity=%d, Price=%f", req.ProductID, req.Quantity, req.Price)
	redisKey := fmt.Sprintf("cart:%d", req.ClientId)
	var cartItems []CartItem

	cartData, err := r.redisClient.Client.Get(ctx, redisKey).Result()
	if err != nil {
		if errors.Is(err, redis2.Nil) {
			log.Printf("Cart not found for Client ID: %d", req.ClientId)
			cartItems = []CartItem{}
		} else {
			return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to get cart from Redis: %w", err)
		}
	} else {
		if err := json.Unmarshal([]byte(cartData), &cartItems); err != nil {
			return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to parse cart data: %w", err)
		}
	}

	if req.CartId == 0 {
		log.Printf("Creating cart for Client ID: %d", req.ClientId)
		cartResp, err := r.CreateCartIfNotExists(ctx, CreateCartIfNotExistsRequest{
			ClientId: req.ClientId,
		})
		if err != nil {
			return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to create or retrieve cart: %w", err)
		}
		req.CartId = cartResp.CartId
	}

	var productPrice float64
	err = r.db.QueryRowContext(ctx, "SELECT price FROM products WHERE id = $1", req.ProductID).Scan(&productPrice)
	if err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to retrieve product price: %w", err)
	}

	log.Printf("Adding item to cart: ProductID=%d, Quantity=%d, Price=%f", req.ProductID, req.Quantity, productPrice)
	updateOrAddItem(&cartItems, req.ProductID, req.Quantity, productPrice)

	updateCartData, err := json.Marshal(cartItems)
	if err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to marshal cart items: %w", err)
	}
	log.Printf("Updating cart data in Redis: %s", updateCartData)
	result1 := r.redisClient.Client.Set(ctx, redisKey, updateCartData, 0)
	if err := result1.Err(); err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to save cart to Redis: %w", err)
	}

	result, err := r.db.ExecContext(ctx, AddItemToCartSQL, req.CartId, req.ProductID, req.Quantity)
	if err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to add item to cart: %w", err)
	}

	log.Printf("Successfully saved cart to Redis for Client ID: %d with key: %s", req.ClientId, redisKey)

	affectedRows, _ := result.RowsAffected()
	if affectedRows == 0 {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("not enough quantity in stock")
	}

	log.Printf("Item successfully added to cart for Client ID: %d", req.ClientId)
	return AddItemToCartResponse{Success: true}, nil
}

func updateOrAddItem(cartItems *[]CartItem, productID int32, quantity int32, price float64) {
	for i, item := range *cartItems {
		if item.ProductID == productID {
			(*cartItems)[i].ProductQuantity += quantity
			return
		}
	}
	*cartItems = append(*cartItems, CartItem{
		ProductID:       productID,
		ProductQuantity: quantity,
		ProductPrice:    price,
	})
}

func (r *UserRepository) DeleteItemFromCart(ctx context.Context, req DeleteItemFromCartRequest) (resp DeleteItemFromCartResponse, err error) {
	log.Printf("Initial Client ID: %d", req.ClientId)
	redisKey := fmt.Sprintf("cart:%d", req.ClientId)
	var cartItems []CartItem

	cartData, err := r.redisClient.Client.Get(ctx, redisKey).Result()
	if err != nil {
		if errors.Is(err, redis2.Nil) {
			log.Printf("Cart not found for Client ID: %d", req.ClientId)
			cartItems = []CartItem{}
		} else {
			return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("failed to get cart from Redis: %w", err)
		}
	} else {
		if err := json.Unmarshal([]byte(cartData), &cartItems); err != nil {
			return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("failed to parse cart data: %w", err)
		}
	}

	updatedItems, itemFound := decreaseItemQuantity(cartItems, req.ProductID)

	if !itemFound {
		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("item not found in cart")
	}

	updateCartData, err := json.Marshal(updatedItems)
	if err != nil {
		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("failed to marshal cart items: %w", err)
	}
	result1 := r.redisClient.Client.Set(ctx, redisKey, updateCartData, 0)
	if err := result1.Err(); err != nil {
		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("failed to save cart to Redis: %w", err)
	}

	err = r.db.QueryRowContext(ctx, GetCartSQL, req.ClientId).Scan(&req.CartId)
	if err != nil {
		if err == sql.ErrNoRows {
			return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("cart not found for user_id %d", req.ClientId)
		}
		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("failed to find cart_id: %v", err)
	}

	result, err := r.db.ExecContext(context.Background(), DeleteItemFromCartSQL2, req.CartId, req.ProductID)
	if err != nil {
		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("failed to delete item from cart: %w", err)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("failed to check affected rows: %w", err)
	}
	if affectedRows == 0 {
		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("no items were updated or deleted")
	}

	return DeleteItemFromCartResponse{Success: true}, nil
}

func decreaseItemQuantity(cartItems []CartItem, productID int32) ([]CartItem, bool) {
	for i, item := range cartItems {
		if item.ProductID == productID {
			if item.ProductQuantity > 1 {
				item.ProductQuantity--
				cartItems[i] = item
				return cartItems, true
			} else {
				return append(cartItems[:i], cartItems[i+1:]...), true
			}
		}
	}
	return cartItems, false
}

func (r *UserRepository) GetCart(ctx context.Context, req GetCartRequest) (resp GetCartResponse, err error) {
	redisKey := fmt.Sprintf("cart:%d", req.ClientId)

	log.Printf("Attempting to fetch cart from Redis with key: %s", redisKey)
	cartData, err := r.redisClient.Client.Get(ctx, redisKey).Result()
	if err != nil {
		if errors.Is(err, redis2.Nil) {
			log.Printf("Cart not found for Client ID: %d", req.ClientId)
		} else {
			return GetCartResponse{}, fmt.Errorf("failed to get cart from Redis: %w", err)
		}
	} else {
		log.Printf("Cart data retrieved from Redis for Client ID %d: %s", req.ClientId, cartData)
		if err := json.Unmarshal([]byte(cartData), &resp.CartItems); err != nil {
			return GetCartResponse{}, fmt.Errorf("failed to parse cart data: %w", err)
		}
	}

	log.Printf("Cart items after unmarshal: %+v", resp.CartItems)

	totalPrice := 0.0

	if len(resp.CartItems) == 0 {
		log.Printf("Cart not found in Redis, fetching from database for Client ID: %d", req.ClientId)

		err = r.db.QueryRowContext(ctx, GetCartSQL, req.ClientId).Scan(&req.CartId)
		if err != nil {
			if err == sql.ErrNoRows {
				return GetCartResponse{}, fmt.Errorf("cart not found for user_id %d", req.ClientId)
			}
			return GetCartResponse{}, fmt.Errorf("failed to find cart_id for user_id %d: %v", req.ClientId, err)
		}

		log.Printf("Found CartId for user_id %d: %d", req.ClientId, req.CartId)

		rows, err := r.db.QueryContext(ctx, GetCartItemSQL, req.CartId)
		if err != nil {
			return GetCartResponse{}, fmt.Errorf("failed to get cart items for cart_id %d: %w", req.CartId, err)
		}
		defer func() {
			if closeErr := rows.Close(); closeErr != nil {
				err = fmt.Errorf("failed to close rows: %w", closeErr)
			}
		}()

		log.Printf("Fetched rows for cart_id %d", req.CartId)

		resp.CartItems = []CartItem{}

		for rows.Next() {
			var cartItem CartItem
			if err := rows.Scan(&cartItem.ProductID, &cartItem.ProductQuantity, &cartItem.ProductPrice); err != nil {
				return resp, fmt.Errorf("failed to scan product for cart_id %d: %w", req.CartId, err)
			}

			productPrice := cartItem.ProductPrice
			log.Printf("Scanned item: ProductID=%d, Quantity=%d, Price=%.2f", cartItem.ProductID, cartItem.ProductQuantity, productPrice)

			resp.CartItems = append(resp.CartItems, cartItem)
			totalPrice += productPrice * float64(cartItem.ProductQuantity)
		}

		if err := rows.Err(); err != nil {
			return resp, fmt.Errorf("error occurred during row iteration for cart_id %d: %w", req.CartId, err)
		}

		updateCartData, err := json.Marshal(resp.CartItems)
		if err != nil {
			return GetCartResponse{}, fmt.Errorf("failed to marshal cart items: %w", err)
		}
		if err := r.redisClient.Client.Set(ctx, redisKey, updateCartData, 0).Err(); err != nil {
			return GetCartResponse{}, fmt.Errorf("failed to save cart to Redis: %w", err)
		}
	} else {
		for _, item := range resp.CartItems {
			productPrice := float64(item.ProductPrice)
			totalPrice += productPrice * float64(item.ProductQuantity)
		}
	}

	resp.TotalPrice = fmt.Sprintf("%.2f", totalPrice)
	return resp, nil
}

func (r *UserRepository) SimulatePayment(ctx context.Context, req PaymentRequest) (resp PaymentResponse, err error) {

	result, err := r.db.ExecContext(ctx, PaymentSQL, req.ClientId)

	if err != nil {
		return resp, fmt.Errorf("ошибка обработки платежа: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return resp, fmt.Errorf("ошибка проверки затронутых строк: %v", err)
	}
	if rowsAffected == 0 {
		return resp, fmt.Errorf("платёж не выполнен: возможно, недостаточно средств на счёте")
	}

	redisKey := fmt.Sprintf("cart:%d", req.ClientId)
	emptyCartData, err := json.Marshal([]CartItem{})
	if err != nil {
		return resp, fmt.Errorf("ошибка сериализации пустой корзины: %v", err)
	}

	err = r.redisClient.Client.Set(ctx, redisKey, emptyCartData, 0).Err()
	if err != nil {
		log.Printf("Ошибка очистки содержимого корзины в Redis для Client ID %d: %v", req.ClientId, err)
	} else {
		log.Printf("Содержимое корзины успешно очищено в Redis для Client ID %d", req.ClientId)
	}

	resp = PaymentResponse{Success: true}
	return resp, nil

}
