// Copyright 2016 Openprovider Authors. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/openprovider/spawn"
	"github.com/openprovider/spawn/auth"
	"github.com/takama/daemon"
)

const (
	// Version - application version
	Version = spawn.VERSION

	// Date - application revision date
	Date = spawn.DATE
)

// Service used for manage daemon and config
type Service struct {
	*Config
	daemon.Daemon
}

// New - creates a new service record
func newService(name, description string) (*Service, error) {
	daemonInstance, err := daemon.New(name, description)
	if err != nil {
		return nil, err
	}

	return &Service{newConfig(), daemonInstance}, nil
}

// Run - manages the service
func (service *Service) Run() (string, error) {

	// if received any kind of command, do it
	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install(os.Args[2:]...)
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		}
	}

	// Load configuration
	if err := service.Load(); err != nil {
		return "Loading config was unsuccessful", err
	}

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	serviceHostPort := fmt.Sprintf("%s:%d", service.Host, service.Port)
	apiHostPort := fmt.Sprintf("%s:%d", service.API.Host, service.API.Port)
	server, err := spawn.NewServer(Description)
	if err != nil {
		return "Initialize service:", err
	}
	// Initialize auth service
	authService, err := auth.NewAuth(&service.AuthEngine)
	if err != nil {
		return "Initialize authentication:", err
	}
	status, err := server.Run(
		serviceHostPort,
		apiHostPort,
		nil,
		service.Nodes,
		service.QueryMode.RoundRobin,
		service.QueryMode.ByPriority,
		service.Check,
		authService,
	)
	if err != nil {
		return status, err
	}
	stdlog.Println(status)

	// Logs a which is host&port used
	stdlog.Printf("%s started on %s\n", Description, serviceHostPort)
	stdlog.Printf("API loaded on %s\n", apiHostPort)

	// loop work cycle with accept connections or interrupt
	// by system signal
	for {
		select {
		case killSignal := <-interrupt:
			stdlog.Println("Got signal:", killSignal)
			stdlog.Println("Stoping listening on ", serviceHostPort, apiHostPort)
			status, err := server.Shutdown()
			if err != nil {
				return status, err
			}
			if killSignal == os.Interrupt {
				stdlog.Println("Service was interruped by system signal")
			} else {
				stdlog.Println("Service was killed")
			}
			return status, nil
		}
	}

	// never happen, but need to complete code
	return "If you see that, you are lucky bastard", nil
}
