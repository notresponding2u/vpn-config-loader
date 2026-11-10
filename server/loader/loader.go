package loader

import (
	"fmt"
	"os/exec"
	"syscall"

	"github.com/notresponding2u/vpn-config-loader/server/config"
)

type Loader struct {
	cmd *exec.Cmd
}

func NewLoader(conf *config.Config) *Loader {
	return &Loader{
		cmd: exec.Command(
			conf.OpenVPNCommand,
			conf.OpenVPNAttribute,
			conf.OpenVPNConfigFile,
		),
	}
}

func (l *Loader) StartVpn() error {
	go func() {
		if err := l.cmd.Run(); err != nil {
			fmt.Println("can't start openvpn: %s", err.Error())
		}

	}()

	return nil
}

func (l *Loader) RestartVpn() error {
	if err := l.cmd.Process.Signal(syscall.SIGINT); err != nil {
		return fmt.Errorf("interupt signal eror: %s", err.Error())
	}

	if err := l.StartVpn(); err != nil {
		return fmt.Errorf("can't restart vpn: %s", err.Error())
	}

	return nil
}
