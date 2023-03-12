package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aakrasnova/zone-mate/bot"
	"github.com/aakrasnova/zone-mate/service"
	"github.com/aakrasnova/zone-mate/storage"
)

type config struct {
	TgToken string `json:"tg_token"`
}

func readCfg(path string) (*config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var c config
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, err
	}

	return &c, nil
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	cfgPath := "./cfg.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	cfg, err := readCfg(cfgPath)
	if err != nil {
		return err
	}

	storage, err := storage.NewStorage("./db.db")
	if err != nil {
		return err
	}
	defer storage.Close()

	service := service.NewService(storage, 500)

	b, err := bot.NewBot(service, cfg.TgToken)
	if err != nil {
		return err
	}
	go b.Run()

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)

	<-terminate
	b.Stop()

	return nil
}
