package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/notresponding2u/vpn-config-loader/server/config"
	"github.com/notresponding2u/vpn-config-loader/server/manager"
)

const (
	OpenVPNConfigFilesDir = "/app/conf.d"
	OpenVPNConfigFileDir = "/vpn/conf"
)

type server struct {
	Conf       *config.Config
	vpnManager *manager.Manager
	logger     *slog.Logger
}

func NewServer(conf *config.Config, logger *slog.Logger, vpnManager *manager.Manager) *server {
	return &server{
		Conf:       conf,
		vpnManager: vpnManager,
		logger:     logger,
	}
}

func (s *server) ListConfigs(endpoint string, engine *gin.Engine) {
	engine.GET(endpoint, func(c *gin.Context) {
		r, err := os.ReadDir(OpenVPNConfigFilesDir)
		if err != nil {
			s.logger.Error("can't read directory", "err", err.Error())
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

		file := fmt.Sprintf("%s/%s", OpenVPNConfigFilesDir, conf)
		dat, err := os.ReadFile(file)
		if err != nil {
			c.String(http.StatusNotFound, "config not found")

			s.logger.Error("can't load file", "file", file, "err", err.Error())
			return
		}

		_, err = os.Stat(OpenVPNConfigFileDir)

		if os.IsNotExist(err) {
			os.Mkdir(OpenVPNConfigFileDir, 0600)
		}

		err = os.WriteFile(
			"/vpn/conf/config.ovpn",
			dat,
			0644,
		)

		if err != nil {
			s.logger.Error("error writing to the file", "err", err.Error())
			c.String(http.StatusInternalServerError, err.Error())

			return
		}

		s.logger.Debug("file loaded")

		if err = s.vpnManager.RestartVpn(); err != nil {
			s.logger.Error("can't load config", "err", err.Error())
			c.String(http.StatusInternalServerError, "can't load config")

			return
		}

		c.Status(http.StatusCreated)
	})
}

func (s *server) GetStatus(endpoint string, engine *gin.Engine) {
	engine.GET(endpoint, func(c *gin.Context) {
		s, err := s.vpnManager.Status()
		if err != nil {
			c.String(http.StatusInternalServerError, "can't get OpenVPN status")

			return
		}

		c.String(http.StatusOK, *s)

	})
}
