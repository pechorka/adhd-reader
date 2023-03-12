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

// var zoneDirs = []string{
// 	// Update path according to your OS
// 	"/usr/share/zoneinfo/",
// 	"/usr/share/lib/zoneinfo/",
// 	"/usr/lib/locale/TZ/",
// }

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
	cfg, err := readCfg("./cfg.json")
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

// func ReadFile(zoneDir, path string) []string {
// 	fileNames := []string{}
// 	files, err := ioutil.ReadDir(filepath.Join(zoneDir, path))
// 	if err != nil {
// 		fmt.Println("failed to read dir", filepath.Join(zoneDir, path), "err: ", err)
// 		return fileNames
// 	}
// 	for _, f := range files {
// 		if f.Name() != strings.ToUpper(f.Name()[:1])+f.Name()[1:] {
// 			continue
// 		}
// 		fullName := filepath.Join(path, f.Name())
// 		if f.IsDir() {
// 			fileNames = append(fileNames, ReadFile(zoneDir, fullName)...)
// 		} else {
// 			fileNames = append(fileNames, fullName)
// 		}
// 	}
// 	return fileNames
// }
