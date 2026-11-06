package main

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/gin-gonic/gin"
	"github.com/notresponding2u/vpn-config-loader/config"
	"github.com/notresponding2u/vpn-config-loader/loader"
)

const (
	// Load these wiht env.
	OpenVPNCommand    = "openvpn"
	OpenVPNAttribute  = "--config"
	OpenVPNConfigFile = "/app/client.ovpn"
)

func main() {
	conf, err := config.New()
	if err != nil {
		fmt.Printf("can't load config: %s\n", err.Error())
	}

	if err := launchOpenVPN(conf); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	l := loader.NewLoader(conf)

	l.ListConfigs("/configs", router)
	l.LoadConfig("/load", router)

	router.Run(":8080")
}

func launchOpenVPN(conf *config.Config) error {
	if _, err := os.Stat(OpenVPNConfigFile); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("file does not exist: %s", OpenVPNConfigFile)
	}

	cmd := exec.Command(conf.OpenVPNCommand, conf.OpenVPNAttribute, conf.OpenVPNConfigFile)

	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("command execution error: %s", err.Error())
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %s\n", err)
	}
	fmt.Println("OpenVPN launched")

	return nil
}
