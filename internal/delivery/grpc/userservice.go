package grpc

import (
	"context"
	"fmt"
	"github.com/Dmitrij-bot/marketserv/internal/usecase"
	pb "github.com/Dmitrij-bot/marketserv/proto"
	"log"
	"strconv"
)

type UserService struct {
	useCase *usecase.UserUseCase
	pb.UnimplementedUserServiceServer
}

func NewUserService(u *usecase.UserUseCase) *UserService {
	return &UserService{
		useCase: u,
	}
}

func (s *UserService) FindClientByUsername(ctx context.Context, req *pb.FindClientByUsernameRequest) (*pb.FindClientByUsernameResponse, error) {

	log.Printf("Received FindClientByUsername request: %v", req)

	id, err := strconv.Atoi(req.Id)
	if err != nil {
		log.Printf("Error converting ID to int: %v", err)
		return nil, err
	}

	userResp, err := s.useCase.FindClientByUsername(ctx, usecase.FindClientByUsernameRequest{
		ClientID: id,
	})
	if err != nil {
		log.Printf("Error finding user: %v", err)
		return nil, err
	}

	resp := &pb.FindClientByUsernameResponse{
		Id:       strconv.Itoa(userResp.ClientID),
		Username: userResp.Username,
		Role:     userResp.Role,
	}

	log.Printf("User found: %v", resp)

	return resp, nil
}

func (s *UserService) SearchProductByName(ctx context.Context, req *pb.SearchProductByNameRequest) (*pb.SearchProductByNameResponse, error) {
	log.Printf("Received SearchProductByName request: %v", req)

	if req.Name == "" {
		return nil, fmt.Errorf("product name cannot be empty")
	}

	productResp, err := s.useCase.SearchProductByName(ctx, usecase.SearchProductByNameRequest{
		ProductName: req.Name,
	})
	if err != nil {
		log.Printf("Error finding name: %v", err)
		return nil, err
	}
	var products []*pb.Product
	for _, p := range productResp.Products {
		products = append(products, &pb.Product{
			Id:          p.ProductID,
			Name:        p.ProductName,
			Description: p.ProductDescription,
			Price:       p.ProductPrice,
		})
	}

	resp := &pb.SearchProductByNameResponse{
		Products: products,
	}

	log.Printf("Product found: %v", resp)

	return resp, nil
}

func (s *UserService) CreateCartIfNotExists(ctx context.Context, req *pb.AddToCartRequest) (*pb.AddToCartResponse, error) {
	log.Printf("Received CreateCartIfNotExists request: %v", req)

	if req.UserId == 0 {
		return nil, fmt.Errorf("user id cannot be zero")
	}

	cartResp, err := s.useCase.CreateCartIfNotExists(
		ctx,
		usecase.CreateCartIfNotExistsRequest{
			ClientId: int(req.UserId),
		})
	if err != nil {
		log.Printf("Error creating or retrieving cart: %v", err)
		return nil, fmt.Errorf("failed to create or retrieve cart: %w", err)
	}

	resp := &pb.AddToCartResponse{
		Message: fmt.Sprintf("Cart created or retrieved successfully with ID: %d", cartResp.CartId),
	}

	log.Printf("Cart creation response: %v", resp)

	return resp, nil
}

func (s *UserService) AddItemToCart(ctx context.Context, req *pb.AddToCartRequest) (*pb.AddToCartResponse, error) {
	log.Printf("Received AddItemToCart request: %v", req)

	if req.UserId == 0 || req.ProductId == 0 || req.Quantity == 0 {
		return nil, fmt.Errorf("invalid input: userId, productId, and quantity must be greater than zero")
	}

	_, err := s.useCase.AddItemToCart(
		ctx,
		usecase.AddItemToCartRequest{
			CartId:    int(req.UserId),
			ProductID: req.ProductId,
			Quantity:  int(req.Quantity),
		})

	if err != nil {
		log.Printf("Error adding item to cart: %v", err)
		return nil, fmt.Errorf("failed to add item to cart: %w", err)
	}

	resp := &pb.AddToCartResponse{
		Message: fmt.Sprintf("Item with product ID %d added to cart successfully", req.ProductId),
	}

	return resp, nil

}
