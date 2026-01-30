package main

import (
	"echotest/internal/app"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/alecthomas/kingpin/v2"
)

var (
	configpath = kingpin.Flag("config", "config file path").Short('f').String()
)

func main() {
	kingpin.Version("0.0.0")
	kingpin.Parse()

	configPath := *configpath
	if configPath == "" {
		log.Println("use default config.yaml")
		exePath, err := os.Executable()
		if err != nil {
			panic(fmt.Sprintf("获取程序路径失败: %v", err))
		}
		configPath = filepath.Join(filepath.Dir(exePath), "config", "config.yaml")
	}

	a, err := app.InitApp(configPath)
	if err != nil {
		panic(err)
	}

	if err := a.Run(); err != nil {
		a.E.Logger.Error("failed to shutdown server", "error", err)
		os.Exit(1)
	}
}
