package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/notresponding2u/vpn-config-loader/server/config"
	"github.com/notresponding2u/vpn-config-loader/server/manager"
	"github.com/notresponding2u/vpn-config-loader/server/server"
)

func main() {
	conf, err := config.New()
	if err != nil {
		fmt.Printf("can't load config: %s\n", err.Error())
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: conf.LogLevel,
	}))

	router := gin.Default()

	vpnManager := manager.New(conf, logger)

	if err := vpnManager.StartVpn(); err != nil {
		logger.Error("can't launch openvpn", "err", err.Error())
		os.Exit(1)
	}

	s := server.NewServer(conf, logger, vpnManager)

	s.ListConfigs("/configs", router)
	s.LoadConfig("/load", router)

	router.Run(":8080")
}
