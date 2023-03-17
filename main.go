package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pechorka/adhd-reader/bot"
	"github.com/pechorka/adhd-reader/queue"
	"github.com/pechorka/adhd-reader/service"
	"github.com/pechorka/adhd-reader/storage"
)

type config struct {
	TgToken string `json:"tg_token"`
	Debug   bool   `json:"debug"`
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

	var store *storage.Storage
	if cfg.Debug {
		store, err = storage.NewTempStorage()
	} else {
		store, err = storage.NewStorage("./db.db")
	}
	if err != nil {
		return err
	}
	defer store.Close()

	service := service.NewService(store, 500)
	msgQueue := queue.NewMessageQueue(queue.Config{})
	b, err := bot.NewBot(service, msgQueue, cfg.TgToken)
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
