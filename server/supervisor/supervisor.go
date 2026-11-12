package supervisor

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

type SupervisorCommand int

const (
	VPNStart SupervisorCommand = iota
	VPNRestart
	VPNReconnect
)

type Supervisor struct {
	conf    *config.Config
	done    chan struct{}
	command chan SupervisorCommand
}

func New(conf *config.Config) *Supervisor {
	fmt.Println("building new supervisor")
	return &Supervisor{
		conf:    conf,
		done:    make(chan struct{}, 10),
		command: make(chan SupervisorCommand, 10),
	}
}

func (s *Supervisor) Stop() {
	s.done <- struct{}{}
}

func (s *Supervisor) Run() (err error) {
	fmt.Println("start iniitiated")

	addr := fmt.Sprintf("%s:%s", s.conf.OpenVPNAttributeManagementIP, s.conf.OpenVPNAttributeManagementPort)

	go func() {
		_, err = s.startVpn()
		if err != nil {
			err = fmt.Errorf("can't start vpn: %s", err.Error())

			return
		}

		for {
			select {
			case <-s.done:
				fmt.Println("stopping supervisor")
			case <-s.command:
				go func() {
					channel, events, err := connectManagement(addr)
					if err != nil {
						err = fmt.Errorf("can't connect to management: %s", err.Error())
						s.done <- struct{}{}
					}
					defer channel.Close()

					if err = s.restartVpn(channel, events); err != nil {
						err = fmt.Errorf("can't restart vpn: %s", err.Error())
						s.done <- struct{}{}
					}
				}()
			}
		}
	}()

	return nil
}

func (s *Supervisor) Restart() error {
	s.command <- VPNRestart

	return nil
}

func (s *Supervisor) startVpn() (*exec.Cmd, error) {
	fullPath := fmt.Sprintf("%s/%s", s.conf.OpenVPNConfigFileDir, s.conf.OpenVPNConfigFile)
	if _, err := os.Stat(fullPath); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("file does not exist: %s", fullPath)
	}

	cmd := exec.Command(
		s.conf.OpenVPNCommand,
		s.conf.OpenVPNAttributeConfig,
		fullPath,
		s.conf.OpenVPNAttributeManagement,
		s.conf.OpenVPNAttributeManagementIP,
		s.conf.OpenVPNAttributeManagementPort,
	)

	startScanners(cmd)

	fmt.Println("starting OpenVPN process...")

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("command execution error: %s", err.Error())
	}

	fmt.Println("OpenVPN launched")

	return cmd, nil
}

func (s *Supervisor) restartVpn(channel *openvpn.MgmtClient, events chan openvpn.Event) error {
	if err := channel.SetStateEvents(true); err != nil {
		return fmt.Errorf("enabling state events: %w", err)
	}

	fmt.Println("listening to incomming events initiating")

	go func(events chan openvpn.Event) {
		for event := range events {
			fmt.Printf("OpenVPN event: %v\n", event)
		}
		fmt.Println("event channel closed")
	}(events)

	done := make(chan error, 1)

	go func() {
		done <- channel.SendSignal("SIGHUP")
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

func startScanners(cmd *exec.Cmd) {
	fmt.Println("starting scanners")

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

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
}

func connectManagement(addr string) (*openvpn.MgmtClient, chan openvpn.Event, error) {
	fmt.Println("dialing vpn")

	events := make(chan openvpn.Event, 100)

	client, err := openvpn.Dial(addr, events)
	if err != nil {
		return nil, nil, err
	}

	fmt.Println("dailing vpn complete")

	return client, events, nil
}
