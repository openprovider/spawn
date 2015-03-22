// Copyright 2015 Igor Dolzhikov. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package spawn

import (
	"strings"

	"github.com/takama/router"
)

func displayAllMethods(c *router.Control) {
	methods := []string{
		headerOfMethods,
		listNodeMethods(),
	}
	desc := strings.Join(methods, "")
	c.Body(desc)
}

func displayNodeMethods(c *router.Control) {
	c.Body(headerOfMethods + listNodeMethods())
}

func listNodeMethods() string {
	return nodeMethods
}

var headerOfMethods = `
Spawn Sync Service

The Spawn service used as a HTTP REST sync service, that makes clustering mode
simpler and easier for most of applications.
`
var nodeMethods = `
Get node settings by host and port

+-------------+-----------+----------------------+
| Method      | Operation | URL                  |
+-------------+-----------+----------------------+
| Get Node    | GET       | /nodes/:host/:port   |
+-------------+-----------+----------------------+

+-----------------+------------------+
| Parameter       | Type             |
+-----------------+------------------+
| host            | string           |
| port            | number           |
| priority        | number           |
| active          | boolean          |
| maintenance     | boolean          |
+-----------------+------------------+


Set node settings by host and port

+-------------+-----------+----------------------+
| Method      | Operation | URL                  |
+-------------+-----------+----------------------+
| Set Node    | PUT       | /nodes/:host/:port   |
+-------------+-----------+----------------------+

+-----------------+------------------+----------+----------------+
| Parameter       | Type             | Required | Default values |
+-----------------+------------------+----------+----------------+
| host            | string           | yes      |                |
| port            | number           | yes      |                |
| priority        | number           | no       | 0              |
| active          | boolean          | no       | false          |
| maintenance     | boolean          | no       | false          |
+-----------------+------------------+----------+----------------+
`
