package manager

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/NordSecurity/gopenvpn/openvpn"
	"github.com/notresponding2u/vpn-config-loader/server/config"
)

type Manager struct {
	conf           *config.Config
	logger         *slog.Logger
	managerAddress string
}

func New(conf *config.Config, logger *slog.Logger) *Manager {
	return &Manager{
		conf:           conf,
		logger:         logger,
		managerAddress: fmt.Sprintf("%s:%s", conf.OpenVPNAttributeManagementIP, conf.OpenVPNAttributeManagementPort),
	}
}

func (m *Manager) RestartVpn() error {
	client, err := m.connectManagement()
	if err != nil {
		return fmt.Errorf("can't connect to management: %s", err.Error())
	}
	defer client.Close()

	m.logger.Debug("sending restart signal")

	done := make(chan error, 1)

	go func() {
		done <- client.SendSignal("SIGHUP")
	}()

	select {
	case err := <-done:
		m.logger.Debug("SendSignal completed", "err", err)

	case <-time.After(10 * time.Second):
		m.logger.Debug("SendSignal is still blocked after 10 seconds")
	}

	m.logger.Debug("restart signal sent")

	return nil
}

func (m *Manager) connectManagement() (*openvpn.MgmtClient, error) {
	events := make(chan openvpn.Event, 100)

	client, err := openvpn.Dial(m.managerAddress, events)
	if err != nil {
		return nil, err
	}

	if err := client.SetStateEvents(true); err != nil {
		return nil, fmt.Errorf("enabling state events: %w", err)
	}

	m.logger.Debug("listening to incomming events initiating")

	go func() {
		for event := range events {
			m.logger.Debug("OpenVPN event: %v\n", event)
		}
		m.logger.Debug("event channel closed")
	}()

	return client, nil
}

func (m *Manager) Status() (*string, error) {
	m.logger.Debug("getting status")

	client, err := m.connectManagement()
	if err != nil {
		return nil, fmt.Errorf("can't connect to management: %s", err)
	}
	defer client.Close()

	done := make(chan error, 1)

	s := ""
	go func() {
		// Retrieve status.
		state, err := client.LatestState()
		if err != nil {
			err = fmt.Errorf("error retreiving status: %s", err.Error())
			return
		}

		s = state.NewState()
		done <- nil
	}()

	select {
	case err := <-done:
		m.logger.Debug("getting status completed", "err", err)
	case <-time.After(10 * time.Second):
		m.logger.Debug("getting status is blocked after 10 seconds")
		return nil, errors.New("gettint status timed out")

	}

	return &s, err
}
