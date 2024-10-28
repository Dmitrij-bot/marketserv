package config

import (
	"encoding/json"
	"github.com/Dmitrij-bot/marketserv/internal/grpc"
	"github.com/Dmitrij-bot/marketserv/pkg/postgres"
	"os"
)

type Config struct {
	GRPC     grpc.Config
	Postgres postgres.Config
}

func Load(filepath string) (cfg Config, err error) {

	file, err := os.Open(filepath)
	if err != nil {
		return cfg, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
