package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/pechorka/adhd-reader/internal/handler"
	"github.com/pechorka/adhd-reader/internal/handler/mw/auth"
	"github.com/pechorka/adhd-reader/internal/service"
	"github.com/pechorka/adhd-reader/internal/storage"

	"github.com/pechorka/adhd-reader/cmd/tgbot/internal/bot"
	"github.com/pechorka/adhd-reader/pkg/encryptor"
	"github.com/pechorka/adhd-reader/pkg/fileloader"
	"github.com/pechorka/adhd-reader/pkg/i18n"
	"github.com/pechorka/adhd-reader/pkg/queue"
	"github.com/pechorka/adhd-reader/pkg/watcher"
	"github.com/pechorka/adhd-reader/pkg/webscraper"
)

// todo move to config
const (
	defaulChunkSize    = 500
	defaultMaxFileSize = 50 * 1024 * 1024 // 20 MB
)

// todo migrate to .env
type config struct {
	Port    int     `json:"port"`
	TgToken string  `json:"tg_token"`
	Debug   bool    `json:"debug"`
	DbPath  string  `json:"db_path"`
	Admins  []int64 `json:"admins"`
	Secret  string  `json:"secret"`
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
	if c.Port == 0 {
		c.Port = 8080
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
	i18nPath := "./i18n.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		i18nPath = os.Args[2]
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

	i18nService := i18n.New()
	watcher, err := watcher.LoadAndWatch(i18nPath, i18nService)
	if err != nil {
		return err
	}
	defer watcher.Close()

	scrapper := webscraper.New()
	encryptor := encryptor.NewEncryptor(cfg.Secret)
	service := service.NewService(store, 500, scrapper, encryptor)
	msgQueue := queue.NewMessageQueue(queue.Config{})
	fileLoader := fileloader.NewLoader(fileloader.Config{
		MaxFileSize: defaultMaxFileSize,
	})
	b, err := bot.NewBot(bot.Config{
		Token:       cfg.TgToken,
		Service:     service,
		MsgQueue:    msgQueue,
		FileLoader:  fileLoader,
		I18n:        i18nService,
		MaxFileSize: defaultMaxFileSize,
		AdminUsers:  cfg.Admins,
	})
	if err != nil {
		return err
	}
	go b.Run()

	handlers := handler.NewHandlers(service)
	mx := chi.NewRouter()
	authMW := auth.NewAuthMW(service)
	mx.With(authMW.Auth)
	mx.Route("/api/v1", func(r chi.Router) {
		handlers.Register(r)
	})

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)

	<-terminate
	b.Stop()

	return nil
}
