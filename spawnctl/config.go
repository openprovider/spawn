// Copyright 2015 Openprovider Authors. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"os"

	"github.com/takama/spawn"
)

// Default values: path to config file, host, port, etc
const (
	defaultConfigPath   = "spawn.conf"
	defaultHost         = "0.0.0.0"
	defaultPort         = 7117
	defaultAPIHost      = "0.0.0.0"
	defaultAPIPort      = 7118
	defaultCheckSec     = 10
	defaultCheckURL     = "/"
	defaultCheckPattern = ""
)

// Config - Application configuration
type Config struct {
	Path string `json:-`

	ShowVersion bool `json:-`

	Host string `json:"host"`
	Port int    `json:"port"`

	QueryMode struct {
		RoundRobin bool `json:"round-robin"`
		ByPriority bool `json:"by-priority"`
	} `json:"query-mode"`

	Check spawn.HealthCheck `json:"health-check"`

	API struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"api"`

	TestMode bool `json:"testMode"`

	Nodes []spawn.Node `json:"nodes"`
}

// New - returns new config record initialized with default values
func newConfig() *Config {
	config := new(Config)
	flag.BoolVar(&config.ShowVersion, "version", false, "show version")
	flag.BoolVar(&config.ShowVersion, "v", false, "show version")
	flag.BoolVar(&config.TestMode, "t", config.TestMode, "")
	flag.BoolVar(&config.TestMode, "test", config.TestMode, "")
	flag.StringVar(&config.Path, "config",
		defaultConfigPath, "path to configuration file")
	flag.StringVar(&config.Host, "host", defaultHost, "host name or IP address")
	flag.IntVar(&config.Port, "port", defaultPort, "port number")
	flag.BoolVar(&config.QueryMode.RoundRobin, "round-robin",
		config.QueryMode.RoundRobin, "use round-robin mode for querying of nodes")
	flag.BoolVar(&config.QueryMode.ByPriority, "by-priority",
		config.QueryMode.ByPriority, "nodes will queried according to priority")
	flag.DurationVar(&config.Check.Seconds, "check-sec",
		defaultCheckSec, "check nodes every number of seconds")
	flag.StringVar(&config.Check.URL, "check-url",
		defaultCheckURL, "url to check node")
	flag.StringVar(&config.Check.Pattern, "check-regexp",
		defaultCheckPattern, "regexp pattern to check node")
	flag.StringVar(&config.API.Host, "api-host",
		defaultAPIHost, "API host name or IP address")
	flag.IntVar(&config.API.Port, "api-port", defaultPort, "API port number")

	return config
}

// Load settings from config file or from sh command line
func (config *Config) Load() error {
	var path string
	var err error

	if err = config.loadConfigFile(config.Path); err != nil {
		return err
	}

	// overwrite config from file by cmd flags
	flags := flag.NewFlagSet("spawn", flag.ContinueOnError)
	// Begin ignored flags
	flags.StringVar(&path, "config", "", "")
	// End ignored flags
	flags.BoolVar(&config.TestMode, "t", config.TestMode, "")
	flags.BoolVar(&config.TestMode, "test", config.TestMode, "")
	flags.StringVar(&config.Host, "host", config.Host, "")
	flags.IntVar(&config.Port, "port", config.Port, "")
	flags.BoolVar(&config.QueryMode.RoundRobin, "round-robin",
		config.QueryMode.RoundRobin, "")
	flags.BoolVar(&config.QueryMode.ByPriority, "by-priority",
		config.QueryMode.ByPriority, "")
	flags.DurationVar(&config.Check.Seconds, "check-sec", config.Check.Seconds, "")
	flags.StringVar(&config.Check.URL, "check-url", config.Check.URL, "")
	flags.StringVar(&config.Check.Pattern, "check-regexp", config.Check.Pattern, "")
	flags.StringVar(&config.API.Host, "api-host", config.API.Host, "")
	flags.IntVar(&config.API.Port, "api-port", config.API.Port, "")
	flags.Parse(os.Args[1:])

	return nil
}

// LoadConfigFile - loads congig file into config record
func (config *Config) loadConfigFile(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := json.NewDecoder(bufio.NewReader(file)).Decode(&config); err != nil {
		return err
	}

	return nil
}
