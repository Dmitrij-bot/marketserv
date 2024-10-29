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

func (s *UserService) AddItemToCart(ctx context.Context, req *pb.AddToCartRequest) (*pb.AddToCartResponse, error) {
	log.Printf("Received AddItemToCart request: user_id: %d, product_id: %d, quantity: %d", req.UserId, req.ProductId, req.Quantity)

	if req.UserId == 0 || req.ProductId == 0 || req.Quantity == 0 {
		return nil, fmt.Errorf("invalid input: userId, productId, and quantity must be greater than zero")
	}

	_, err := s.useCase.AddItemToCart(
		ctx,
		usecase.AddItemToCartRequest{
			ClientId:  req.UserId,
			ProductID: req.ProductId,
			Quantity:  req.Quantity,
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

func (s *UserService) DeleteItemFromCart(ctx context.Context, req *pb.DeleteFromCartRequest) (*pb.DeleteFromCartResponse, error) {
	log.Printf("Receved DeleteItemFromCart : user_id: %d, product_id: %d", req.UserId, req.ProductId)
	if req.UserId == 0 || req.ProductId == 0 {
		return nil, fmt.Errorf("invalid input: userId, productId must be greater than zero")
	}

	_, err := s.useCase.DeleteItemFromCart(
		ctx,
		usecase.DeleteItemFromCartRequest{
			ClientId:  req.UserId,
			ProductID: req.ProductId,
		})

	if err != nil {
		log.Printf("Error delete item from cart: %v", err)
		return nil, fmt.Errorf("failed to delete item from cart: %w", err)
	}
	resp := &pb.DeleteFromCartResponse{
		Message: fmt.Sprintf("Item with product ID %d delete from cart successfully", req.ProductId),
	}
	return resp, nil
}

func (s *UserService) GetCart(ctx context.Context, req *pb.GetCartRequest) (*pb.GetCartResponse, error) {

	cartResp, err := s.useCase.GetCart(ctx, usecase.GetCartRequest{
		ClientId: req.UserId,
	})
	if err != nil {
		log.Printf("failed to get cart for user_id %d: %v", req.UserId, err)
		return nil, fmt.Errorf("failed to get cart for user_id %d: %v", req.UserId, err)
	}
	log.Printf("Usecase returned cart: Items - %v, TotalPrice - %s", cartResp.CartItems, cartResp.TotalPrice)
	var cartItems []*pb.CartItem
	for _, item := range cartResp.CartItems {
		cartItems = append(cartItems, &pb.CartItem{
			ProductId: item.ProductID,
			Quantity:  strconv.Itoa(int(item.ProductQuantity)),
			Price:     item.ProductPrice,
		})
	}

	log.Printf("Returning response: Items - %v, TotalPrice - %s", cartItems, cartResp.TotalPrice)

	resp := &pb.GetCartResponse{
		Items:      cartItems,
		TotalPrice: cartResp.TotalPrice,
	}
	return resp, nil
}

func (s *UserService) SimulatePayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {

	_, err := s.useCase.SimulatePayment(
		ctx,
		usecase.PaymentRequest{
			ClientId: req.UserId,
		})

	if err != nil {
		return nil, fmt.Errorf("failed to payment: %w", err)
	}

	resp := &pb.PaymentResponse{
		Success: true,
		Message: "Оплата успешно выполнена.",
	}

	return resp, nil
}
