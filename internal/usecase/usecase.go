package usecase

import (
	"context"
	"fmt"
	"github.com/Dmitrij-bot/marketserv/internal/repository"
	"log"
)

type UserUseCase struct {
	r repository.Interface
}

func New(r repository.Interface) *UserUseCase {
	return &UserUseCase{
		r: r,
	}
}

func (u *UserUseCase) FindClientByUsername(ctx context.Context, req FindClientByUsernameRequest) (resp FindClientByUsernameResponse, err error) {

	user, err := u.r.FindClientByUsername(
		ctx,
		repository.FindClientByUsernameRequest{
			ClientID: req.ClientID,
		})
	if err != nil {
		return resp, err
	}

	return FindClientByUsernameResponse{
		ClientID: user.ClientID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

func (u *UserUseCase) SearchProductByName(ctx context.Context, req SearchProductByNameRequest) (resp SearchProductByNameResponse, err error) {

	if req.ProductName == "" {
		return SearchProductByNameResponse{}, fmt.Errorf("product name cannot be empty")
	}

	productsResp, err := u.r.SearchProductByName(
		ctx,
		repository.SearchProductByNameRequest{
			ProductName: req.ProductName,
		})
	if err != nil {
		return SearchProductByNameResponse{}, fmt.Errorf("failed to search products: %w", err)
	}

	var products []Product
	for _, p := range productsResp.Products {
		products = append(products, Product{
			ProductID:          p.ProductID,
			ProductName:        p.ProductName,
			ProductDescription: p.ProductDescription,
			ProductPrice:       p.ProductPrice,
		})
	}

	return SearchProductByNameResponse{
		Products: products,
	}, nil
}

func (u *UserUseCase) AddItemToCart(ctx context.Context, req AddItemToCartRequest) (resp AddItemToCartResponse, err error) {
	log.Printf("Creating cart for Client ID: %d", req.ClientId)
	/*cartResp, err := u.r.CreateCartIfNotExists(ctx, repository.CreateCartIfNotExistsRequest{
		ClientId: req.ClientId,
	})
	if err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to check or create cart: %w", err)
	}*/
	log.Printf("Adding item to cart for Client ID: %d", req.ClientId)
	addResp, err := u.r.AddItemToCart(
		ctx,
		repository.AddItemToCartRequest{
			CartId:    req.CartId,
			ProductID: req.ProductID,
			Quantity:  req.Quantity,
			ClientId:  req.ClientId,
		})

	if err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to add to cart: %w", err)
	}

	return AddItemToCartResponse{
		Success: addResp.Success,
	}, nil
}

func (u *UserUseCase) DeleteItemFromCart(ctx context.Context, req DeleteItemFromCartRequest) (resp DeleteItemFromCartResponse, err error) {

	deleteResp, err := u.r.DeleteItemFromCart(
		ctx,
		repository.DeleteItemFromCartRequest{
			ClientId:  req.ClientId,
			ProductID: req.ProductID,
		})
	if err != nil {
		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("failed to delete from cart: %w", err)
	}

	return DeleteItemFromCartResponse{
		Success: deleteResp.Success,
	}, nil
}

func (u *UserUseCase) GetCart(ctx context.Context, req GetCartRequest) (resp GetCartResponse, err error) {

	if req.ClientId == 0 {
		return GetCartResponse{}, fmt.Errorf("invalid user_id: %d", req.ClientId)
	}
	log.Printf("Received GetCart request for user_id: %d", req.ClientId)

	getResp, err := u.r.GetCart(
		ctx,
		repository.GetCartRequest{
			ClientId: req.ClientId})
	if err != nil {
		log.Printf("Error fetching cart for user_id %d: %v", req.ClientId, err)
		return GetCartResponse{}, fmt.Errorf("usecase: failed to get cart for user_id %d: %v", req.ClientId, err)
	}

	var cartItems []CartItem
	for _, repoItem := range getResp.CartItems {
		cartItems = append(cartItems, CartItem{
			ProductID:       repoItem.ProductID,
			ProductQuantity: repoItem.ProductQuantity,
			ProductPrice:    repoItem.ProductPrice,
		})
	}

	log.Printf("Returning from GetCart: CartItems - %v, TotalPrice - %s", cartItems, getResp.TotalPrice)

	return GetCartResponse{
		CartItems:  cartItems,
		TotalPrice: getResp.TotalPrice,
	}, nil
}

func (u *UserUseCase) SimulatePayment(ctx context.Context, req PaymentRequest) (resp PaymentResponse, err error) {

	paymentResp, err := u.r.SimulatePayment(
		ctx,
		repository.PaymentRequest{
			ClientId: req.ClientId,
		})
	if err != nil {
		return PaymentResponse{Success: false}, fmt.Errorf("failed to payment: %w", err)
	}

	return PaymentResponse{
		Success: paymentResp.Success,
	}, nil
}
