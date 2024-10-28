package app

import (
	"context"
	"fmt"
	"github.com/Dmitrij-bot/marketserv/config"
	"github.com/Dmitrij-bot/marketserv/internal/delivery/grpc"
	grpc2 "github.com/Dmitrij-bot/marketserv/internal/grpc"
	"github.com/Dmitrij-bot/marketserv/internal/repository"
	"github.com/Dmitrij-bot/marketserv/internal/usecase"
	"github.com/Dmitrij-bot/marketserv/pkg/lyfecycle"
	"github.com/Dmitrij-bot/marketserv/pkg/postgres"
	"log"
)

type App struct {
	cfg  config.Config
	cmps []cmp
}
type cmp struct {
	Service lyfecycle.Lyfecycle
	Name    string
}

func New(cfg config.Config) *App {
	return &App{cfg: cfg}
}

func (app *App) Start(ctx context.Context) error {
	db := postgres.NewDB(app.cfg.Postgres)
	userRepo := repository.NewUserRepository(db)
	userUseCase := usecase.New(userRepo)
	userService := grpc.NewUserService(userUseCase)
	grpcServer := grpc2.NewGRPCServer(app.cfg.GRPC, userService)

	app.cmps = append(
		app.cmps,
		cmp{db, "grpc db"},
		cmp{grpcServer, "grpcServ"},
	)

	okCh, errCh := make(chan struct{}), make(chan error)

	go func() {
		for _, c := range app.cmps {
			log.Printf("%v is starting", c.Name)

			if err := c.Service.Start(ctx); err != nil {
				err = fmt.Errorf("Cannot start %s: %v", c.Name, err)

				log.Println(err)

				errCh <- err

				return
			}

			log.Printf("%v started", c.Name)
		}
		okCh <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout gg")
	case err := <-errCh:
		return err
	case <-okCh:
		log.Printf("Application started!")
		return nil
	}
}

func (app *App) Stop(ctx context.Context) error {
	log.Println("shutting down service...")
	okCh, errCh := make(chan struct{}), make(chan error)

	go func() {
		for i := len(app.cmps) - 1; i > 0; i-- {
			c := app.cmps[i]
			log.Println("stopping %q...", c.Name)

			if err := c.Service.Stop(ctx); err != nil {
				log.Println(err)
				errCh <- err

				return
			}
		}

		okCh <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout gg")
	case err := <-errCh:
		return err
	case <-okCh:
		log.Println("Application stopped!")
		return nil
	}
}
