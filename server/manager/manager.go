package manager

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/NordSecurity/gopenvpn/openvpn"
	"github.com/notresponding2u/vpn-config-loader/server/config"
)

type Manager struct {
	conf   *config.Config
	logger *slog.Logger
}

func New(conf *config.Config, logger *slog.Logger) *Manager {
	return &Manager{
		conf:   conf,
		logger: logger,
	}
}

func (m *Manager) StartVpn() error {
	fullPath := fmt.Sprintf("%s/%s", m.conf.OpenVPNConfigFileDir, m.conf.OpenVPNConfigFile)
	if _, err := os.Stat(fullPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("file does not exist: %s", fullPath)
	}

	cmd := exec.Command(
		"openvpn",
		"--config",
		fullPath,
		"--management",
		m.conf.OpenVPNAttributeManagementIP,
		m.conf.OpenVPNAttributeManagementPort,
	)

	m.startScanners(cmd)

	m.logger.Debug("starting OpenVPN process...")

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("command execution error: %s", err.Error())
	}

	m.logger.Debug("OpenVPN launched")

	return nil
}

func (m *Manager) RestartVpn() error {
	addr := fmt.Sprintf("%s:%s", m.conf.OpenVPNAttributeManagementIP, m.conf.OpenVPNAttributeManagementPort)

	client, events, err := connectManagement(addr)
	if err != nil {
		return fmt.Errorf("can't connect to management: %s", err.Error())
	}
	defer client.Close()

	if err := client.SetStateEvents(true); err != nil {
		return fmt.Errorf("enabling state events: %w", err)
	}

	m.logger.Debug("listening to incomming events initiating")

	go func() {
		for event := range events {
			m.logger.Debug("OpenVPN event: %v\n", event)
		}
		m.logger.Debug("event channel closed")
	}()

	done := make(chan error, 1)

	m.logger.Debug("sending restart signal")

	go func() {
		done <- client.SendSignal("SIGHUP")
	}()

	select {
	case err := <-done:
		m.logger.Debug("SendSignal completed: %v\n", err)

	case <-time.After(10 * time.Second):
		m.logger.Debug("SendSignal is still blocked after 10 seconds")
	}

	m.logger.Debug("restart signal sent")

	return nil
}

func connectManagement(addr string) (*openvpn.MgmtClient, chan openvpn.Event, error) {
	events := make(chan openvpn.Event, 100)

	client, err := openvpn.Dial(addr, events)
	if err != nil {
		return nil, nil, err
	}

	return client, events, nil
}

func (m *Manager) startScanners(cmd *exec.Cmd) {
	m.logger.Debug("starting scanners")

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			m.logger.Debug("[OpenVPN]", "msg", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			m.logger.Error("scanner error", "err", err)
		}

	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			m.logger.Error("[OpenVPN ERROR]", "err", scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			m.logger.Error("scanner error", "err", err)
		}

	}()
}
