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

type TxHandler func(tx *sql.Tx) error

func NewUserRepository(db *postgres.DB, redisClient *redis.RedisDB) *UserRepository {
	return &UserRepository{
		db:          db,
		redisClient: redisClient,
	}
}

func (r *UserRepository) withTransaction(ctx context.Context, handler TxHandler) error {

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback()
			log.Println("rollback:", err)
			if rollbackErr != nil {
				err = fmt.Errorf("rollback failed: %w, original error: %v", rollbackErr, err)
			}
			return
		}

		commitErr := tx.Commit()
		if commitErr != nil {
			err = fmt.Errorf("commit failed: %w", commitErr)
		}
	}()

	err = handler(tx)
	if err != nil {
		return err
	}

	return nil
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

	err = r.withTransaction(ctx, func(tx *sql.Tx) error {

		rows, err := tx.QueryContext(ctx, SearchProductByNameSQL, req.ProductName)
		if err != nil {
			return fmt.Errorf("failed to query products: %w", err)
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
				return fmt.Errorf("failed to scan product: %w", err)
			}
			resp.Products = append(resp.Products, product)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("error occurred during row iteration: %w", err)
		}

		if len(resp.Products) == 0 {
			return fmt.Errorf("no products found for name: %s", req.ProductName)
		}

		searchEvent := SearchProductEvent{
			ProductName: req.ProductName,
			Products:    resp.Products,
			Message:     fmt.Sprintf("Продукты по запросу '%s' успешно найдены", req.ProductName),
		}

		if err := r.SaveKafkaMessage2(
			ctx,
			tx,
			SaveKafkaMessageRequest{
				KafkaMessage: searchEvent,
				KafkaKey:     SearchProductEventKey,
			}); err != nil {
			log.Printf("Ошибка сохранения сообщения: %v", err)
		} else {
			log.Printf("событие сохранено: %v", SaveKafkaMessageRequest{KafkaMessage: searchEvent})
		}

		return nil
	})

	if err != nil {
		return SearchProductByNameResponse{}, err
	}

	return resp, nil
}

func (r *UserRepository) CreateCartIfNotExists(ctx context.Context, req CreateCartIfNotExistsRequest) (resp CreateCartIfNotExistsResponse, err error) {

	err = r.db.QueryRowContext(ctx, GetCartSQL, req.ClientId).Scan(&resp.CartId)

	if err == sql.ErrNoRows {
		log.Printf("Cart not found for Client ID: %d, creating new cart", req.ClientId)

		err := r.db.QueryRowContext(ctx, CreateCartIfNotExistsSQL, req.ClientId).Scan(&resp.CartId)
		if err != nil {
			return resp, fmt.Errorf("failed to create cart: %w", err)
		}
		log.Printf("Successfully created cart for Client ID: %d, Cart ID: %d", req.ClientId, resp.CartId)
	} else if err != nil {
		return resp, fmt.Errorf("failed to retrieve cart: %w", err)
	}
	log.Printf("Cart found for Client ID: %d, Cart ID: %d", req.ClientId, resp.CartId)
	return resp, nil
}

func (r *UserRepository) AddItemToCart(ctx context.Context, req AddItemToCartRequest) (resp AddItemToCartResponse, err error) {

	tx, err := r.db.Begin()
	if err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback()
			log.Println("rollback:", err)
			if rollbackErr != nil {
				err = fmt.Errorf("rollback failed: %w, original error: %v", rollbackErr, err)
			}
			return
		}

		commitErr := tx.Commit()
		if commitErr != nil {
			err = fmt.Errorf("commit failed: %w", commitErr)
		}
	}()

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
		log.Printf("Searching  cart for Client ID: %d", req.ClientId)
		cartResp, err := r.CreateCartIfNotExists(ctx, CreateCartIfNotExistsRequest{
			ClientId: req.ClientId,
		})
		if err != nil {
			return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to create or retrieve cart: %w", err)
		}
		req.CartId = cartResp.CartId
	}

	var productPrice float64
	var stockQuantity int32
	err = tx.QueryRowContext(ctx, "SELECT price, quantity FROM products WHERE id = $1", req.ProductID).Scan(&productPrice, &stockQuantity)
	if err == sql.ErrNoRows {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("product with ID %d not found", req.ProductID)
	} else if err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to retrieve product data: %w", err)
	}

	if stockQuantity < req.Quantity {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("not enough quantity in stock for product ID %d", req.ProductID)
	}

	updateOrAddItem(&cartItems, req.ProductID, req.Quantity, productPrice)

	updateCartData, err := json.Marshal(cartItems)
	if err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to marshal cart items: %w", err)
	}
	defer func() {
		if err == nil {
			log.Printf("Updating cart data in Redis: %s", updateCartData)
			result1 := r.redisClient.Client.Set(ctx, redisKey, updateCartData, 0)
			if err := result1.Err(); err != nil {
				log.Printf("failed to save cart to Redis: %v", err)
			}
		}
	}()

	result, err := tx.ExecContext(ctx, AddItemToCartSQL, req.CartId, req.ProductID, req.Quantity)
	if err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to add item to cart: %w", err)
	}

	affectedRows, _ := result.RowsAffected()
	if affectedRows == 0 {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("not enough quantity in stock")
	}

	addEvent := AddEvent{
		ClientID:  int(req.ClientId),
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Message: fmt.Sprintf("Товар успешно добавлен в корзину {\"client_id\":%d,\"product_id\":%d,\"quantity\":%d}",
			req.ClientId, req.ProductID, req.Quantity),
	}

	if err := r.SaveKafkaMessage2(
		ctx,
		tx,
		SaveKafkaMessageRequest{
			KafkaMessage: addEvent,
			KafkaKey:     AddEventKey,
		}); err != nil {
		log.Printf("Ошибка сохранения сообщения: %v", err)
	} else {
		log.Printf("событие сохранено: %v", SaveKafkaMessageRequest{KafkaMessage: addEvent})
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

	err = r.withTransaction(ctx, func(tx *sql.Tx) error {
		redisKey := fmt.Sprintf("cart:%d", req.ClientId)
		var cartItems []CartItem

		cartData, err := r.redisClient.Client.Get(ctx, redisKey).Result()
		if err != nil {
			if errors.Is(err, redis2.Nil) {
				log.Printf("Cart not found for Client ID: %d", req.ClientId)
				cartItems = []CartItem{}
			} else {
				return fmt.Errorf("failed to get cart from Redis: %w", err)
			}
		} else {
			if err := json.Unmarshal([]byte(cartData), &cartItems); err != nil {
				return fmt.Errorf("failed to parse cart data: %w", err)
			}
		}

		updatedItems, itemFound := decreaseItemQuantity(cartItems, req.ProductID)

		if !itemFound {
			return fmt.Errorf("item not found in cart")
		}

		updateCartData, err := json.Marshal(updatedItems)
		if err != nil {
			return fmt.Errorf("failed to marshal updated cart items: %w", err)
		}

		defer func() {
			if err == nil {
				log.Printf("Updating cart data in Redis: %s", updateCartData)
				result1 := r.redisClient.Client.Set(ctx, redisKey, updateCartData, 0)
				if err := result1.Err(); err != nil {
					log.Printf("failed to save cart to Redis: %v", err)
				}
			}
		}()

		err = tx.QueryRowContext(ctx, GetCartSQL, req.ClientId).Scan(&req.CartId)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("cart not found for user_id %d", req.ClientId)
			}
			return fmt.Errorf("failed to find cart_id: %v", err)
		}

		result, err := tx.ExecContext(ctx, DeleteItemFromCartSQL2, req.CartId, req.ProductID)
		if err != nil {
			return fmt.Errorf("failed to delete item from cart in database: %w", err)
		}

		affectedRows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to check affected rows: %w", err)
		}
		if affectedRows == 0 {
			return fmt.Errorf("no items were updated or deleted in the database")
		}

		deleteEvent := DeleteEvent{
			ClientID: int(req.ClientId),
			Message: fmt.Sprintf("Товар успешно удален из корзины {\"client_id\":%d,\"product_id\":%d}",
				req.ClientId, req.ProductID),
		}

		if err := r.SaveKafkaMessage2(
			ctx,
			tx,
			SaveKafkaMessageRequest{
				KafkaMessage: deleteEvent,
				KafkaKey:     DeleteEventKey,
			}); err != nil {
			log.Printf("Ошибка сохранения сообщения: %v", err)
		} else {
			log.Printf("событие сохранено: %v", SaveKafkaMessageRequest{KafkaMessage: deleteEvent})
		}

		return nil
	})

	if err != nil {
		return DeleteItemFromCartResponse{Success: false}, err
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

	err = r.withTransaction(ctx, func(tx *sql.Tx) error {
		redisKey := fmt.Sprintf("cart:%d", req.ClientId)

		cartData, err := r.redisClient.Client.Get(ctx, redisKey).Result()
		if err != nil {
			if errors.Is(err, redis2.Nil) {
				log.Printf("Cart not found for Client ID: %d", req.ClientId)
			} else {
				return fmt.Errorf("failed to get cart from Redis: %w", err)
			}
		} else {
			log.Printf("Cart data retrieved from Redis for Client ID %d: %s", req.ClientId, cartData)
			if err := json.Unmarshal([]byte(cartData), &resp.CartItems); err != nil {
				return fmt.Errorf("failed to parse cart data: %w", err)
			}
		}
		totalPrice := 0.0
		for _, item := range resp.CartItems {
			totalPrice += float64(item.ProductQuantity) * item.ProductPrice
		}
		resp.TotalPrice = fmt.Sprintf("%.2f", totalPrice)

		if len(resp.CartItems) == 0 {
			log.Printf("Cart not found in Redis, fetching from database for Client ID: %d", req.ClientId)

			err = tx.QueryRowContext(ctx, GetCartSQL, req.ClientId).Scan(&req.CartId)
			if err != nil {
				if err == sql.ErrNoRows {
					return fmt.Errorf("cart not found for user_id %d", req.ClientId)
				}
				return fmt.Errorf("failed to find cart_id for user_id %d: %v", req.ClientId, err)
			}

			rows, err := tx.QueryContext(ctx, GetCartItemSQL, req.CartId)
			if err != nil {
				return fmt.Errorf("failed to get cart items for cart_id %d: %w", req.CartId, err)
			}
			defer func() {
				if closeErr := rows.Close(); closeErr != nil {
					err = fmt.Errorf("failed to close rows: %w", closeErr)
				}
			}()

			resp.CartItems = []CartItem{}
			totalPrice := 0.0

			for rows.Next() {
				var cartItem CartItem
				if err := rows.Scan(&cartItem.ProductID, &cartItem.ProductQuantity, &cartItem.ProductPrice); err != nil {
					return fmt.Errorf("failed to scan product for cart_id %d: %w", req.CartId, err)
				}

				productPrice := cartItem.ProductPrice
				totalPrice += productPrice * float64(cartItem.ProductQuantity)
				resp.CartItems = append(resp.CartItems, cartItem)
			}

			if err := rows.Err(); err != nil {
				return fmt.Errorf("error occurred during row iteration for cart_id %d: %w", req.CartId, err)
			}

			resp.TotalPrice = fmt.Sprintf("%.2f", totalPrice)

			updateCartData, err := json.Marshal(resp.CartItems)
			if err != nil {
				return fmt.Errorf("failed to marshal cart items: %w", err)
			}
			if err := r.redisClient.Client.Set(ctx, redisKey, updateCartData, 0).Err(); err != nil {
				log.Printf("failed to save cart to Redis: %w", err)
			}
		}

		getcartEvent := GetCartEvent{
			ClientID:   int(req.ClientId),
			CartItems:  resp.CartItems,
			TotalPrice: resp.TotalPrice,
			Message:    fmt.Sprintf("Корзина для клиента %d успешно получена", req.ClientId),
		}

		if err := r.SaveKafkaMessage2(
			ctx,
			tx,
			SaveKafkaMessageRequest{
				KafkaMessage: getcartEvent,
				KafkaKey:     GetCartEventKey,
			}); err != nil {
			log.Printf("Ошибка сохранения сообщения: %v", err)
		} else {
			log.Printf("событие сохранено: %v", SaveKafkaMessageRequest{KafkaMessage: getcartEvent})
		}

		return nil
	})

	if err != nil {
		return GetCartResponse{}, err
	}

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

func (r *UserRepository) SaveKafkaMessage(ctx context.Context, req SaveKafkaMessageRequest) (resp SaveKafkaMessageResponse, err error) {

	kafkaMessage, err := json.Marshal(req.KafkaMessage)
	if err != nil {
		return resp, fmt.Errorf("failed to serialize Kafka message: %w", err)
	}

	query := `
		INSERT INTO events (kafka_key, kafka_message, status, created_at)
		VALUES ($1, $2, DEFAULT, DEFAULT)
	`

	_, err = r.db.ExecContext(ctx, query, req.KafkaKey, kafkaMessage)
	if err != nil {
		return SaveKafkaMessageResponse{Success: false}, fmt.Errorf("failed to save Kafka message: %w", err)
	}

	return SaveKafkaMessageResponse{Success: true}, nil
}

func (r *UserRepository) GetKafkaMessage() (resp GetKafkaMessageResponse, err error) {

	row := r.db.QueryRow("SELECT id, kafka_key, kafka_message FROM events WHERE status = 'new' LIMIT 1")

	err = row.Scan(&resp.ID, &resp.Key, &resp.Message)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetKafkaMessageResponse{}, nil
		}
		return GetKafkaMessageResponse{}, fmt.Errorf(" %s, %w", err)
	}

	return GetKafkaMessageResponse{
		ID:      resp.ID,
		Key:     resp.Key,
		Message: resp.Message,
	}, nil
}

func (r *UserRepository) SetDone(id int) error {

	query := "UPDATE events SET status = 'done' WHERE id = $1"
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to execute update for id %d: %w", id, err)
	}

	return nil

}

func (r *UserRepository) SaveKafkaMessage2(ctx context.Context, tx *sql.Tx, req SaveKafkaMessageRequest) error {

	kafkaMessage, err := json.Marshal(req.KafkaMessage)
	if err != nil {
		return fmt.Errorf("failed to serialize Kafka message: %w", err)
	}

	query := `
		INSERT INTO events (kafka_key, kafka_message, status, created_at)
		VALUES ($1, $2, DEFAULT, DEFAULT)
	`

	_, err = tx.ExecContext(ctx, query, req.KafkaKey, kafkaMessage)
	if err != nil {
		return fmt.Errorf("failed to save Kafka message: %w", err)
	}

	return nil
}
