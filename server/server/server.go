package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/notresponding2u/vpn-config-loader/server/loader"
	"github.com/notresponding2u/vpn-config-loader/server/config"
)

type server struct {
	Conf *config.Config
	loader *loader.Loader
}

func NewServer(conf *config.Config,loader *loader.Loader) *server {
	return &server{
		Conf: conf,
		loader: loader,
	}
}

func (s *server) ListConfigs(endpoint string, engine *gin.Engine) {
	engine.GET(endpoint, func(c *gin.Context) {
		r, err := os.ReadDir(s.Conf.OpenVPNConfigFilesDir)
		if err != nil {
			fmt.Printf("can't read directory: %s", err.Error())
			c.Status(http.StatusBadRequest)
		}

		res := []string{}

		for i := range r {
			res = append(res, r[i].Name())
		}

		c.JSON(http.StatusOK, res)
	})
}

func (s *server) LoadConfig(endpoint string, engine *gin.Engine) {
	engine.POST(endpoint, func(c *gin.Context) {
		conf := c.PostForm("config")

		file := fmt.Sprintf("%s/%s", s.Conf.OpenVPNConfigFilesDir, conf)
		dat, err := os.ReadFile(file)
		if err != nil {
			c.String(http.StatusNotFound, "config not found")

			fmt.Printf("can't load file: %s\n%s\n", file, err.Error())
			return
		}

		_, err = os.Stat(s.Conf.OpenVPNConfigFileDir)

		if os.IsNotExist(err) {
			os.Mkdir(s.Conf.OpenVPNConfigFileDir, 0600)
		}

		err = os.WriteFile(
			fmt.Sprintf("%s/%s", s.Conf.OpenVPNConfigFileDir, s.Conf.OpenVPNConfigFile),
			fmt.Appendf(dat, "\n\n%s", s.Conf.OpenVPNConfigAuthFileAddition),
			0644,
		)

		if err != nil {
			fmt.Printf("error writing to the file: %s\n", err.Error())
			c.String(http.StatusBadRequest, err.Error())

			return
		}

		fmt.Printf("file loaded: %s/%s\n",s.Conf.OpenVPNConfigFileDir,s.Conf.OpenVPNConfigFile)

		if err = s.loader.RestartVpn(); err != nil {
			c.String(http.StatusInternalServerError, "can't load config")
			fmt.Printf("can't load config :%s", err.Error())

			return
		}

		c.Status(http.StatusCreated)
	})
}
