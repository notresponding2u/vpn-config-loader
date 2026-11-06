package loader

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/gin-gonic/gin"
	"github.com/notresponding2u/vpn-config-loader/config"
)

type loader struct {
	Conf *config.Config
}

func NewLoader(conf *config.Config) *loader {
	return &loader{
		Conf: conf,
	}
}

func (l *loader) ListConfigs(endpoint string, engine *gin.Engine) {
	engine.GET(endpoint, func(c *gin.Context) {
		r, err := os.ReadDir(l.Conf.OpenVPNConfigFilesDir)
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

func (l *loader) LoadConfig(endpoint string, engine *gin.Engine) {
	engine.POST(endpoint, func(c *gin.Context) {
		conf := c.PostForm("config")

		file := fmt.Sprintf("%s/%s", l.Conf.OpenVPNConfigFilesDir, conf)
		dat, err := os.ReadFile(file)
		if err != nil {
			c.String(http.StatusNotFound, "config not found")

			fmt.Printf("can't load file: %s\n%s\n", file, err.Error())
			return
		}

		_, err = os.Stat(l.Conf.OpenVPNConfigFileDir)

		if os.IsNotExist(err) {
			os.Mkdir(l.Conf.OpenVPNConfigFileDir, 0777)
		}

		err = os.WriteFile(
			fmt.Sprintf("%s/%s", l.Conf.OpenVPNConfigFileDir, l.Conf.OpenVPNConfigFile),
			fmt.Appendf(dat, "\n\n%s", l.Conf.OpenVPNConfigAuthFileAddition),
			0644,
		)

		if err != nil {
			fmt.Printf("error writing to the file: %s\n", err.Error())
			c.String(http.StatusBadRequest, err.Error())

			return
		}

		c.Status(http.StatusCreated)
	})
}

func (l *loader)reloadOpenVPN()error{
//	cmd, err:= exec.Command("")

	return nil
}
