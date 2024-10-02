package grpc

import (
	"context"
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
