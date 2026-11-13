package main

import (
	"fmt"
	"net/http"
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

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	vpnManager := manager.New(conf)

	if err := vpnManager.StartVpn(); err != nil {
		fmt.Printf("can't launch openvpn: %s\n", err.Error())
		os.Exit(1)
	}

	s := server.NewServer(conf, vpnManager)

	s.ListConfigs("/configs", router)
	s.LoadConfig("/load", router)

	router.Run(":8080")
}
