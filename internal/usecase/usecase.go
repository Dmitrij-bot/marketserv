package usecase

import (
	"context"
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
	product, err := u.r.SearchProductByName(
		ctx,
		repository.SearchProductByNameRequest{
			ProductName: req.ProductName,
		})
	if err != nil {
		return resp, err
	}
	return SearchProductByNameResponse{
		ProductID:          product.ProductID,
		ProductName:        product.ProductName,
		ProductDescription: product.ProductDescription,
		ProductPrice:       product.ProductPrice,
	}, nil
}
