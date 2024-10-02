package config

import (
	"github.com/Dmitrij-bot/marketserv/internal/grpc"
	"github.com/Dmitrij-bot/marketserv/pkg/postgres"
)

type Config struct {
	GRPC     grpc.Config
	Postgres postgres.Config
}
