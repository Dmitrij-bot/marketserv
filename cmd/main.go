package main

import (
	"context"
	"github.com/Dmitrij-bot/marketserv/config"
	"github.com/Dmitrij-bot/marketserv/internal/delivery/grpc"
	grpc2 "github.com/Dmitrij-bot/marketserv/internal/grpc"
	"github.com/Dmitrij-bot/marketserv/internal/repository"
	"github.com/Dmitrij-bot/marketserv/internal/usecase"
	"github.com/Dmitrij-bot/marketserv/pkg/postgres"
	"log"
)

func main() {
	cfg := config.Config{
		GRPC: grpc2.Config{
			Host: ":50051",
		},
		Postgres: postgres.Config{
			DBHost:     "localhost",
			DBPort:     "5432",
			DBUser:     "postgres",
			DBPassword: "39786682",
			DBName:     "first",
			SSLMode:    "disable",
		},
	}

	db := postgres.NewDB(cfg.Postgres)
	if err := db.Start(context.Background()); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer func() {
		if err := db.Stop(context.Background()); err != nil {
			log.Printf("failed to close database connection: %v", err)
		}
	}()

	userRepo := repository.NewUserRepository(db.SQLBD())
	userUseCase := usecase.New(userRepo)
	userService := grpc.NewUserService(userUseCase)
	grpcServer := grpc2.NewGRPCServer(cfg.GRPC, userService)

	if err := grpcServer.Start(context.Background()); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		if err := grpcServer.Stop(context.Background()); err != nil {
			log.Printf("failed to stop server: %v", err) // Логирование ошибки
		}
	}()

	select {}
}
