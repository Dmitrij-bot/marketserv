package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Dmitrij-bot/marketserv/pkg/postgres"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"strconv"
)

type UserRepository struct {
	db *postgres.DB
}

func NewUserRepository(db *postgres.DB) *UserRepository {
	return &UserRepository{db: db}
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

	if req.CartId == 0 {
		cartResp, err := r.CreateCartIfNotExists(ctx, CreateCartIfNotExistsRequest{
			ClientId: req.ClientId,
		})
		if err != nil {
			return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to create or retrieve cart: %w", err)
		}
		req.CartId = cartResp.CartId
	}

	_, err = r.db.ExecContext(ctx, AddItemToCartSQL, req.CartId, req.ProductID, req.Quantity)

	if err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to add item to cart: %w", err)
	}

	return AddItemToCartResponse{Success: true}, nil

}

func (r *UserRepository) DeleteItemFromCart(ctx context.Context, req DeleteItemFromCartRequest) (resp DeleteItemFromCartResponse, err error) {

	err = r.db.QueryRowContext(ctx, GetCartSQL, req.ClientId).Scan(&req.CartId)
	if err != nil {
		if err == sql.ErrNoRows {
			return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("cart not found for user_id %d", req.ClientId)
		}
		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("failed to find cart_id: %v", err)
	}

	var exists bool
	err = r.db.QueryRowContext(ctx, SearchProductByIdSQL, req.CartId, req.ProductID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking if item exists in cart: %v", err)
		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("Error checking if item exists in cart: %v", err)
	}

	if !exists {
		return DeleteItemFromCartResponse{Success: false}, status.Errorf(codes.NotFound, "Item not found in cart")
	}

	_, err = r.db.ExecContext(context.Background(), DeleteItemFromCartSQL, req.CartId, req.ProductID)

	if err != nil {
		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("failed to delete item from cart: %w", err)
	}

	return DeleteItemFromCartResponse{Success: true}, nil
}

func (r *UserRepository) GetCart(ctx context.Context, req GetCartRequest) (resp GetCartResponse, err error) {

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
	totalPrice := 0.0

	for rows.Next() {
		var cartItem CartItem
		var productPrice float64

		if err := rows.Scan(&cartItem.ProductID, &cartItem.ProductQuantity, &cartItem.ProductPrice); err != nil {
			return resp, fmt.Errorf("failed to scan product for cart_id %d: %w", req.CartId, err)
		}

		productPrice, err = strconv.ParseFloat(cartItem.ProductPrice, 64)
		if err != nil {
			log.Printf("Error parsing price for ProductID %d: %v", cartItem.ProductID, err)
			return resp, fmt.Errorf("error parsing price for ProductID %d: %v", cartItem.ProductID, err)
		}

		log.Printf("Scanned item: ProductID=%d, Quantity=%d, Price=%.2f", cartItem.ProductID, cartItem.ProductQuantity, productPrice)

		resp.CartItems = append(resp.CartItems, cartItem)

		totalPrice += productPrice * float64(cartItem.ProductQuantity)
	}

	if err := rows.Err(); err != nil {
		return resp, fmt.Errorf("error occurred during row iteration for cart_id %d: %w", req.CartId, err)
	}

	resp = GetCartResponse{
		CartItems:  resp.CartItems,
		TotalPrice: fmt.Sprintf("%.2f", totalPrice), // Форматируем перед отправкой
	}

	return resp, nil
}
