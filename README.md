Spawn Sync service
==================

[![Build Status](https://travis-ci.org/openprovider/spawn.svg?branch=master)](https://travis-ci.org/openprovider/spawn)
[![GoDoc](https://godoc.org/github.com/openprovider/spawn?status.svg)](https://godoc.org/github.com/openprovider/spawn)

The Spawn service used as a HTTP REST sync service, that makes clustering mode simpler and easier for most
of applications. What's the idea? There are several applications, which are developed to provide their service
through HTTP Rest API. But you have no idea how to provide failover processing and clustering mode for these 
applications, because they are not compatible with, etc. And here we go. The Spawn service will make this
job instead of the whole bunch of services that must be configured and communicating with each other.
How it works? Let's see the scheme:

![Scheme](https://github.com/openprovider/spawn/blob/master/scheme/scheme.png)

All GET requests from the Client will reproduce with a selected node. A node for GET requests will be selected
by specified 'round-robin' and 'by priority' mode if they are active. All updates requests (PUT, POST, DELETE)
will be repeated to every node or will be accumulated in the corresponded queue if the node is unreachable.

If temporarily switch off the node 3 from the process, due to maintenance or network loss, the updates worker
of the node 3 will be stopped automatically and all updates of the node 3 would begin to accumulate in the queue.
After the end of maintenance work all accumulated updates will posted in the node 3. 

Embedded health checker makes possible to recover processing automatically. 

The service has API to control of the nodes. So, if you setup the Spawn API on localhost:7118, you can use query
below to show what nodes are configured:

```sh
curl -XGET http://localhost:7118/nodes?pretty=true

{
  "duration": 25982,
  "took": "25.982µs",
  "data": {
    "results": [
      {
        "host": "node1.myapp.com",
        "port": 7017,
        "priority": 1,
        "active": true,
        "maintenance": false
      },
      {
        "host": "node2.myapp.com",
        "port": 7017,
        "priority": 2,
        "active": true,
        "maintenance": false
      }
    ],
    "success": true,
    "total": 2
  }
}

```

Of course you could modify/delete these setting with PUT/DELETE request. Use API helper '/list' to see all supported methods.

Currently this is not production version, follow the updates.

### Restrictions

- HTTPS is not supported currently


### Config

Example of a config file
```json
{
  "host": "spawn.myapp.com",
  "port": 7117,
  "api": {
    "host": "spawn.myapp.com",
    "port": 7118
  },
  "query-mode": {
    "round-robin": true,
    "by-priority": true
  },
  "health-check": {
    "seconds": 10,
    "url": "/info",
    "regexp": "^version:[0-9]+$"
  },
  "nodes": [
    {
      "host": "node1.myapp.com",
      "port": 7017,
      "priority": 1,
      "active": true,
      "maintenance": false
    },
    {
      "host": "node2.myapp.com",
      "port": 7017,
      "priority": 2,
      "active": true,
      "maintenance": false
    },
    {
      "host": "node3.myapp.com",
      "port": 7017,
      "priority": 3,
      "active": false,
      "maintenance": false
    }
  ]
}
```
### Install from source

You need have installed golang 1.4.2+

This service is "go-gettable", just do:

```sh
go get github.com/openprovider/spawn/spawnctl
```

### Usage

```sh
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
  --by-priority          Nodes will queried according to priority
  --check-sec=SECONDS    Check nodes every number of seconds
  --check-url=URL        URL to check nodes (/info, etc)
  --check-regexp=REGEXP  Regexp pattern to check nodes
```

## Todo

- More of Tests coverage and benchmarks
- Synchronization between the nodes
- Exponential Back-off
- support of HTTPS
- Monitoring system inside
- Admin panel for monitoring and configuration of the nodes
- Extended logging system

## Authors

[Igor Dolzhikov](https://github.com/takama)

## Contributors

All the contributors are welcome. If you would like to be the contributor please accept some rules.
- The pull requests will be accepted only in "develop" branch
- All modifications or additions should be tested

Thank you for your understanding!

## License

[MIT License](https://github.com/openprovider/spawn/blob/master/LICENSE)
