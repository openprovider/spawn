// Copyright 2015 Openprovider Authors. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"
)

const (

	// Name of the service
	Name = "spawn"

	// Description of the service
	Description = "Spawn Sync Service"
)

// Init simplest logger
var (
	stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime)
)

// Init CPU numbers, "Usage" helper
func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Usage = func() {
		fmt.Println(Usage())
	}
}

func main() {
	service, err := newService(Name, Description)
	if err != nil {
		errlog.Println("Error: ", err)
		os.Exit(1)
	}
	flag.Parse()

	if service.ShowVersion {
		buildTime, err := time.Parse(time.RFC3339, Date)
		if err != nil {
			buildTime = time.Now()
		}
		fmt.Println(Description, Version, buildTime.Format(time.RFC3339))
		os.Exit(0)
	}
	status, err := service.Run()
	if err != nil {
		errlog.Println(status, "\nError: ", err)
		os.Exit(1)
	}
	fmt.Println(status)
}
