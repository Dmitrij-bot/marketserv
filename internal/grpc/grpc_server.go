package grpc

import (
	"context"
	"errors"
	grpc2 "github.com/Dmitrij-bot/marketserv/internal/delivery/grpc"
	order "github.com/Dmitrij-bot/marketserv/proto"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	cfg         Config             // Ваша конфигурация
	grpcServer  *grpc.Server       // Указатель на gRPC сервер
	userService *grpc2.UserService // Ваш сервис, реализующий методы gRPC
}

func NewGRPCServer(cfg Config, userService *grpc2.UserService) *Server {
	return &Server{
		cfg:         cfg,
		userService: userService,
	}
}

func (s *Server) Start(ctx context.Context) error {
	if s.grpcServer != nil {
		return errors.New("server is already running")
	}

	s.grpcServer = grpc.NewServer()

	order.RegisterUserServiceServer(s.grpcServer, s.userService)

	reflection.Register(s.grpcServer)

	lis, err := net.Listen("tcp", s.cfg.Host)
	if err != nil {
		return err
	}

	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
		}
	}()

	log.Printf("gRPC server is running on %s", s.cfg.Host)

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.grpcServer == nil {
		return errors.New("server is not running")
	}
	s.grpcServer.GracefulStop()
	return nil
}
