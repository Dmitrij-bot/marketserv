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

/*func (p *repository.Product) ToUseCaseProduct() Product {
	return Product{
		ProductID:          p.ProductID,
		ProductName:        p.ProductName,
		ProductDescription: p.ProductDescription,
		ProductPrice:       p.ProductPrice,
	}
}*/

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
	for _ = range productsResp.Products {
		products = append(products, Product{})
	}

	return SearchProductByNameResponse{
		Products: products,
	}, nil
}
