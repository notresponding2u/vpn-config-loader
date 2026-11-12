package loader

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/NordSecurity/gopenvpn/openvpn"
	"github.com/notresponding2u/vpn-config-loader/server/config"
)

type Loader struct {
	conf             *config.Config
	managementClient *openvpn.MgmtClient
	events           chan openvpn.Event
}

func NewLoader(conf *config.Config) (*Loader, error) {
	client, events, err := connectManagement(fmt.Sprintf("%s:%s", conf.OpenVPNAttributeManagementIP, conf.OpenVPNAttributeManagementPort))
	if err != nil {
		return nil,  fmt.Errorf("can't open management connectin: s%", err.Error())
	}

	return &Loader{
		conf:             conf,
		managementClient: client,
		events:           events,
	}, nil
}

func (l *Loader) CloseManagementConnection() {
	l.managementClient.Close()
}

func (l *Loader) StartVpn() error {
	fullPath := fmt.Sprintf("%s/%s", l.conf.OpenVPNConfigFileDir, l.conf.OpenVPNConfigFile)
	if _, err := os.Stat(fullPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("file does not exist: %s", fullPath)
	}

	cmd := exec.Command(
		l.conf.OpenVPNCommand,
		l.conf.OpenVPNAttributeConfig,
		fullPath,
		l.conf.OpenVPNAttributeManagement,
		l.conf.OpenVPNAttributeManagementIP,
		l.conf.OpenVPNAttributeManagementPort,
	)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	fmt.Println("starting OpenVPN process...")

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("command execution error: %s", err.Error())
	}

	fmt.Println("OpenVPN launched")

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
	if err := l.managementClient.SetStateEvents(true); err != nil {
		return fmt.Errorf("enabling state events: %w", err)
	}

	fmt.Println("listening to incomming events initiating")

	go func() {
		for event := range l.events {
			fmt.Printf("OpenVPN event: %v\n", event)
		}
		fmt.Println("event channel closed")
	}()

	done := make(chan error, 1)

	go func() {
		done <- l.managementClient.SendSignal("SIGHUP")
	}()

	fmt.Println("sending restart signal")
	select {
	case err := <-done:
		fmt.Printf("SendSignal completed: %v\n", err)

	case <-time.After(10 * time.Second):
		fmt.Println("SendSignal is still blocked after 10 seconds")
	}

	fmt.Println("restart signal sent")

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
