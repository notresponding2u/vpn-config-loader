package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/notresponding2u/vpn-config-loader/server/config"
	"github.com/notresponding2u/vpn-config-loader/server/loader"
	"github.com/notresponding2u/vpn-config-loader/server/server"
)

func main() {
	conf, err := config.New()
	if err != nil {
		fmt.Printf("can't load config: %s\n", err.Error())
	}

	//	if err := launchOpenVPN(conf); err != nil {
	//		fmt.Printf("can't launch openvpn: %s", err.Error())
	//		os.Exit(1)
	//	}
	//
	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	l := loader.NewLoader(conf)
	if err := l.StartVpn(); err != nil {
		fmt.Printf("can't launch openvpn: %s", err.Error())
		os.Exit(1)
	}

	s := server.NewServer(conf, l)

	s.ListConfigs("/configs", router)
	s.LoadConfig("/load", router)

	router.Run(":8080")
}

//func launchOpenVPN(conf *config.Config) error {
//	fullPath := fmt.Sprintf("%s/%s", conf.OpenVPNConfigFileDir, conf.OpenVPNConfigFile)
//	if _, err := os.Stat(fullPath); errors.Is(err, os.ErrNotExist) {
//		return fmt.Errorf("file does not exist: %s", fullPath)
//	}
//
//	cmd := exec.Command(conf.OpenVPNCommand, conf.OpenVPNAttribute, conf.OpenVPNConfigFile)
//
//	stderr, _ := cmd.StderrPipe()
//	if err := cmd.Start(); err != nil {
//		return fmt.Errorf("command execution error: %s", err.Error())
//	}
//
//	scanner := bufio.NewScanner(stderr)
//	for scanner.Scan() {
//		fmt.Println(scanner.Text())
//	}
//
//	if err := scanner.Err(); err != nil {
//		return fmt.Errorf("scanner error: %s\n", err)
//	}
//	fmt.Println("OpenVPN launched")
//
//	return nil
//}
