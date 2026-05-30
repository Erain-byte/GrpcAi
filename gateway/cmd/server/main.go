package main

import (
	"flag"
	"gateway/internal/config"
	"log"
)

func main() {
	var configFile string // 定义命令行参数 -f 用于指定配置文件路径，默认为 "etc/user.yaml"
	flag.StringVar(&configFile, "f", "etc/user.yaml", "config file path")
	flag.Parse()
	cfg, err := config.Init(configFile)
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}
}
