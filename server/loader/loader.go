package loader

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/notresponding2u/vpn-config-loader/server/config"
)

type Loader struct {
	conf *config.Config
	cmd  *exec.Cmd
}

func NewLoader(conf *config.Config) *Loader {
	return &Loader{
		conf: conf,
	}
}

func (l *Loader) StartVpn() error {
	fullPath := fmt.Sprintf("%s/%s", l.conf.OpenVPNConfigFileDir, l.conf.OpenVPNConfigFile)
	if _, err := os.Stat(fullPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("file does not exist: %s", fullPath)
	}

	l.cmd = exec.Command(
		l.conf.OpenVPNCommand,
		l.conf.OpenVPNAttribute,
		fullPath,
	)

	stdout, _ := l.cmd.StdoutPipe()
	stderr, _ := l.cmd.StderrPipe()

	if err := l.cmd.Start(); err != nil {
		return fmt.Errorf("command execution error: %s", err.Error())
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Println("[OpenVPN]", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("scanner error: %s\n", err)
		}

	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println("[OpenVPN ERROR]", scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("scanner error: %s\n", err)
		}

	}()

	fmt.Println("OpenVPN launched")

	return nil
}

func (l *Loader) RestartVpn() error {
	if err := l.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("interupt signal eror: %s", err.Error())
	}

	l.cmd.Wait()

	if err := l.StartVpn(); err != nil {
		return fmt.Errorf("can't restart vpn: %s", err.Error())
	}

	return nil
}
