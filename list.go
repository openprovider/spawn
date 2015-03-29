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
		listOfMethods,
	}
	desc := strings.Join(methods, "")
	c.Body(desc)
}

func displayAllNodeMethods(c *router.Control) {
	methods := []string{
		headerOfMethods,
		nodeGetMethods,
		nodeSetMethods,
		nodeDeleteMethods,
	}
	desc := strings.Join(methods, "")
	c.Body(desc)
}

func displayGetNodeMethods(c *router.Control) {
	methods := []string{
		headerOfMethods,
		nodeGetMethods,
	}
	desc := strings.Join(methods, "")
	c.Body(desc)
}
func displaySetNodeMethods(c *router.Control) {
	methods := []string{
		headerOfMethods,
		nodeSetMethods,
	}
	desc := strings.Join(methods, "")
	c.Body(desc)
}
func displayDeleteNodeMethods(c *router.Control) {
	methods := []string{
		headerOfMethods,
		nodeDeleteMethods,
	}
	desc := strings.Join(methods, "")
	c.Body(desc)
}

var headerOfMethods = `
Spawn Sync Service

The Spawn service used as a HTTP REST sync service, that makes
clustering mode simpler and easier for most of applications.
`
var listOfMethods = `
Use helpers to see detailed information about specific methods.

To see all methods of the nodes settings, use:
/list/nodes

To see the methods that get the nodes settings, use:
/list/nodes/get

To see the methods that set the nodes settings, use:
/list/nodes/set

To see the methods that delete the nodes settings, use:
/list/nodes/delete
`
var nodeGetMethods = `
Get node settings specified by host and port
============================================

+----------------+------------------+-------------------------+
| Method         | Operation        | URL                     |
+----------------+------------------+-------------------------+
| Get Node       | GET              | /nodes/:host/:port      |
+----------------+------------------+-------------------------+

+----------------+------------------+-------------------------+
| Parameter      | Type             | Required                |
+----------------+------------------+-------------------------+
| host           | string           | yes                     |
| port           | number           | yes                     |
+----------------+------------------+-------------------------+

Method returns node settings:
+----------------+------------------+-------------------------+
| Data           | Type             | Description             |
+----------------+------------------+-------------------------+
| host           | string           | Host name or IP address |
| port           | number           | Port number             |
| priority       | number           | Priority value          |
| active         | boolean          | Node is active          |
| maintenance    | boolean          | Node is in maintenance  |
+----------------+------------------+-------------------------+

Get nodes settings specified by host
====================================

+----------------+------------------+-------------------------+
| Method         | Operation        | URL                     |
+----------------+------------------+-------------------------+
| Get Nodes      | GET              | /nodes/:host            |
+----------------+------------------+-------------------------+

+----------------+------------------+-------------------------+
| Parameter      | Type             | Required                |
+----------------+------------------+-------------------------+
| host           | string           | yes                     |
+----------------+------------------+-------------------------+

Method returns all nodes settings specified by host:
See description - Get node settings specified by host and port

Get all nodes settings
======================

+----------------+------------------+-------------------------+
| Method         | Operation        | URL                     |
+----------------+------------------+-------------------------+
| Get Nodes      | GET              | /nodes                  |
+----------------+------------------+-------------------------+

Method returns all nodes settings:
See description - Get node settings specified by host and port
`
var nodeSetMethods = `
Set node settings specified by host and port
============================================

+----------------+------------------+-------------------------+
| Method         | Operation        | URL                     |
+----------------+------------------+-------------------------+
| Set Node       | PUT              | /nodes/:host/:port      |
+----------------+------------------+-------------------------+

+----------------+------------------+-------------------------+
| Parameter      | Type             | Required                |
+----------------+------------------+-------------------------+
| host           | string           | yes                     |
| port           | number           | yes                     |
+----------------+------------------+-------------------------+

Method accepts node settings:
+----------------+------------------+-------------------------+---------------+
| Parameter      | Type             | Description             | Default value |
+----------------+------------------+-------------------------+---------------+
| host           | string           | Host name or IP address |               |
| port           | number           | Port number             |               |
| priority       | number           | Priority value          | 0             |
| active         | boolean          | Node is active          | false         |
| maintenance    | boolean          | Node is in maintenance  | false         |
+----------------+------------------+-------------------------+---------------+

Set all nodes settings
======================

+----------------+------------------+-------------------------+
| Method         | Operation        | URL                     |
+----------------+------------------+-------------------------+
| Set Nodes      | PUT              | /nodes                  |
+----------------+------------------+-------------------------+

Method accepts all nodes settings:
See description - Set node settings specified by host and port
`

var nodeDeleteMethods = `
Delete node settings specified by host and port
===============================================

+----------------+------------------+-------------------------+
| Method         | Operation        | URL                     |
+----------------+------------------+-------------------------+
| Delete Node    | DELETE           | /nodes/:host/:port      |
+----------------+------------------+-------------------------+

+----------------+------------------+-------------------------+
| Parameter      | Type             | Required                |
+----------------+------------------+-------------------------+
| host           | string           | yes                     |
| port           | number           | yes                     |
+----------------+------------------+-------------------------+

Method delete node settings specified by host and port

Delete nodes settings specified by host
=======================================

+----------------+------------------+-------------------------+
| Method         | Operation        | URL                     |
+----------------+------------------+-------------------------+
| Delete Nodes   | DELETE           | /nodes/:host            |
+----------------+------------------+-------------------------+

+----------------+------------------+-------------------------+
| Parameter      | Type             | Required                |
+----------------+------------------+-------------------------+
| host           | string           | yes                     |
+----------------+------------------+-------------------------+

Method delete all nodes settings specified by host

Delete all nodes settings
=========================

+----------------+------------------+-------------------------+
| Method         | Operation        | URL                     |
+----------------+------------------+-------------------------+
| Delete Nodes   | DELETE           | /nodes                  |
+----------------+------------------+-------------------------+

Method delete all nodes settings
`
