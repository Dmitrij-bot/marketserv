package main

import (
	"context"
	"github.com/Dmitrij-bot/marketserv/config"
	"github.com/Dmitrij-bot/marketserv/internal/app"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func GetParamFromEnv(env string, defFileName string) string {
	configPath := os.Getenv(env)
	if len(configPath) == 0 {
		configPath = defFileName
	}

	return configPath
}

func main() {

	configPath := GetParamFromEnv("CONFIG", "./config/config.json")

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal("cant load config: " + err.Error())
	}

	a := app.New(cfg)

	startCtx, startCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer startCancel()

	if err := a.Start(startCtx); err != nil {
		log.Fatal(err.Error())
	}

	quitCh := make(chan os.Signal, 1)
	signal.Notify(quitCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	<-quitCh

	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer stopCancel()

	err = a.Stop(stopCtx)
	if err != nil {
		log.Fatal(err.Error())
	}
}
