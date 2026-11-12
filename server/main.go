package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/notresponding2u/vpn-config-loader/server/config"
	"github.com/notresponding2u/vpn-config-loader/server/server"
	"github.com/notresponding2u/vpn-config-loader/server/supervisor"
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

	vpnSupervisor := supervisor.New(conf)

	if err := vpnSupervisor.Run(); err != nil {
		fmt.Printf("can't launch openvpn: %s\n", err.Error())
		os.Exit(1)
	}

	s := server.NewServer(conf, vpnSupervisor)

	s.ListConfigs("/configs", router)
	s.LoadConfig("/load", router)

	router.Run(":8080")
}
