package main

import (
	"encoding/json"
	"fmt"
	"github.com/pechorka/adhd-reader/internal/service"
	"github.com/pechorka/adhd-reader/internal/storage"
	"os"
	"os/signal"
	"syscall"

	"github.com/pechorka/adhd-reader/cmd/tgbot/internal/bot"
	"github.com/pechorka/adhd-reader/pkg/fileloader"
	"github.com/pechorka/adhd-reader/pkg/queue"
)

// todo move to config
const (
	defaulChunkSize    = 500
	defaultMaxFileSize = 20 * 1024 * 1024 // 20 MB
)

// todo migrate to .env
type config struct {
	TgToken string `json:"tg_token"`
	Debug   bool   `json:"debug"`
	DbPath  string `json:"db_path"`
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
	if c.DbPath == "" {
		c.DbPath = "./db.db"
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
		store, err = storage.NewStorage(cfg.DbPath)
	}
	if err != nil {
		return err
	}
	defer store.Close()

	service := service.NewService(store, 500)
	msgQueue := queue.NewMessageQueue(queue.Config{})
	fileLoader := fileloader.NewLoader(fileloader.Config{
		MaxFileSize: defaultMaxFileSize,
	})
	b, err := bot.NewBot(bot.Config{
		Token:       cfg.TgToken,
		Service:     service,
		MsgQueue:    msgQueue,
		FileLoader:  fileLoader,
		MaxFileSize: defaultMaxFileSize,
	})
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