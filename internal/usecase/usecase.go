package usecase

import (
	"context"
	"fmt"
	"github.com/Dmitrij-bot/marketserv/internal/repository"
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

	cartResp, err := u.r.CreateCartIfNotExists(ctx, repository.CreateCartIfNotExistsRequest{
		ClientId: req.ClientId,
	})
	if err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to check or create cart: %w", err)
	}

	addResp, err := u.r.AddItemToCart(
		ctx,
		repository.AddItemToCartRequest{
			CartId:    cartResp.CartId,
			ProductID: req.ProductID,
			Quantity:  req.Quantity,
		})

	if err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to add to cart: %w", err)
	}

	return AddItemToCartResponse{
		Success: addResp.Success,
	}, nil
}
