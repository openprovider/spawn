// Copyright 2016 Openprovider Authors. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package main

var usage = `
spawnctl - Spawn Sync Service Control

Usage:
  spawnctl install | remove | start | stop | status
  spawnctl [ -t | --test ] [ --option | --option ... ]
  spawnctl -h | --help
  spawnctl -v | --version

Commands:
  install           Install as service
  remove            Remove service
  start             Start service
  stop              Stop service
  status            Check service status

  -h --help         Show this screen
  -v --version      Show version
  -t --test         Test mode

Options:
  --config=PATH          Path to the config file
  --host=HOST            Host name or IP address
  --port=PORT            Port number
  --api-host=HOST        API host name or IP address
  --api-port=PORT        API port number
  --round-robin          Use round-robin mode for querying of nodes
  --by-priority          Nodes will used according to priority
  --check-sec=SECONDS    Check nodes every number of seconds
  --check-url=URL        URL to check nodes (/info, etc)
  --check-regexp=REGEXP  Regexp pattern to check nodes
  --auth=TYPE            Auth type (LDAP, oAuth, etc)
  --auth-expire=MINUTES  Auth expiration time (default: 30)
  --auth-host=HOST       Auth service host name or IP address
  --auth-port=PORT       Auth service port number
`

// Usage - get usage information
func Usage() string {
	return usage
}
