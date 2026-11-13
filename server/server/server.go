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

type server struct {
	Conf       *config.Config
	vpnManager *manager.Manager
	logger     *slog.Logger
}

func NewServer(conf *config.Config, logger *slog.Logger, vpnManager *manager.Manager) *server {
	return &server{
		Conf:       conf,
		vpnManager: vpnManager,
		logger: logger,
	}
}

func (s *server) ListConfigs(endpoint string, engine *gin.Engine) {
	engine.GET(endpoint, func(c *gin.Context) {
		r, err := os.ReadDir(s.Conf.OpenVPNConfigFilesDir)
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

		file := fmt.Sprintf("%s/%s", s.Conf.OpenVPNConfigFilesDir, conf)
		dat, err := os.ReadFile(file)
		if err != nil {
			c.String(http.StatusNotFound, "config not found")

			s.logger.Error("can't load file", "file", file, "err", err.Error())
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
			s.logger.Error("error writing to the file", "err", err.Error())
			c.String(http.StatusBadRequest, err.Error())

			return
		}

		s.logger.Debug("file loaded", "dir", s.Conf.OpenVPNConfigFileDir, "file", s.Conf.OpenVPNConfigFile)

		if err = s.vpnManager.RestartVpn(); err != nil {
			c.String(http.StatusInternalServerError, "can't load config")
			s.logger.Error("can't load config", "err", err.Error())

			return
		}

		c.Status(http.StatusCreated)
	})
}
