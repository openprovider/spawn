// Copyright 2015 Openprovider Authors. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package spawn

import (
	"container/ring"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/takama/router"
)

/*
Node contains the node parameters:

- Host is the host name or IP,

- Port is the port number,

- Priority defines a sequence, which will be operating according to attribute of the priority.
  A example of sorted values by priority - from highest to lowest (1,2,3,0,0,0,-1,-2,-3)
  the priority '0' has neutral priority value between high and low,

- Active defines status of the node.
  If it will be set to false, the queue which related with the node will be deleted
  or it will be created otherwise

- Maintenance mode is using to stop the worker and accumulate updates in the queue.
  If maintenance mode set to false all updates will posted in the node.
*/
type Node struct {
	Host        string `json:"host"`
	Port        uint64 `json:"port"`
	Priority    int    `json:"priority"`
	Active      bool   `json:"active"`
	Maintenance bool   `json:"maintenance"`
}

// NodeBundle contains an embedded server link and Node records
type NodeBundle struct {
	// contains filtered or unexported fields
	mutex sync.RWMutex
	*Server
	ring    *ring.Ring
	update  chan nodeJob
	records map[string]map[uint64]Node
}

// nodeJob is struct which contains jobs for update/delete records
type nodeJob struct {
	isDelete bool
	isUpdate bool
	done     bool
	record   Node
}

// byPriority type defines speciallly for sorting by priority attribute
type byPriority []Node

func (bp byPriority) Len() int {
	return len(bp)
}
func (bp byPriority) Swap(i, j int) {
	bp[i], bp[j] = bp[j], bp[i]
}
func (bp byPriority) Less(i, j int) bool {
	if bp[i].Priority > 0 && bp[j].Priority > 0 {
		return bp[i].Priority < bp[j].Priority
	}
	if bp[i].Priority < 0 && bp[j].Priority < 0 {
		return bp[i].Priority > bp[j].Priority
	}
	if bp[i].Priority >= 0 && bp[j].Priority <= 0 {
		return true
	}
	return false
}

// Get - gets one of the node record specified by host and port
func (bundle *NodeBundle) Get(host string, port uint64) (node Node, ok bool) {
	// Lock the bundle for 'read' operation
	bundle.mutex.RLock()
	defer bundle.mutex.RUnlock()

	node, ok = bundle.records[host][port]

	return
}

// GetAllByHost - gets all the nodes records specified by host and sorted according to priority
func (bundle *NodeBundle) GetAllByHost(host string) (nodes []Node, total int) {
	// Lock the bundle for 'read' operation
	bundle.mutex.RLock()
	defer bundle.mutex.RUnlock()

	if _, ok := bundle.records[host]; ok {
		for _, record := range bundle.records[host] {
			nodes = append(nodes, record)
		}
	}
	total = len(nodes)
	if bundle.Server.byPriority {
		sort.Sort(byPriority(nodes))
	}

	return
}

// GetAll - gets all the nodes records sorted according to priority
func (bundle *NodeBundle) GetAll() (nodes []Node, total int) {
	// Lock the bundle for 'read' operation
	bundle.mutex.RLock()
	defer bundle.mutex.RUnlock()

	for host := range bundle.records {
		for _, record := range bundle.records[host] {
			nodes = append(nodes, record)
		}
	}
	total = len(nodes)
	if bundle.Server.byPriority {
		sort.Sort(byPriority(nodes))
	}

	return
}

// Set - updates the node record or create one if it does not exist
func (bundle *NodeBundle) Set(node *Node) bool {

	if node.Host == "" || !isAlphaNumeric(node.Host) || node.Port == 0 {
		return false
	}

	// Add/Update a record
	bundle.update <- nodeJob{isUpdate: true, record: *node}

	// Job done - end of the transaction
	bundle.update <- nodeJob{done: true}
	bundle.job <- nodeJobSignal

	return true
}

// SetAll - updates all the nodes records or create them if records do not exist
func (bundle *NodeBundle) SetAll(nodes []Node) bool {

	// Validate the Nodes
	for _, node := range nodes {
		if node.Host == "" || !isAlphaNumeric(node.Host) || node.Port == 0 {
			return false
		}
	}

	for _, node := range nodes {
		// Add/Update a record
		bundle.update <- nodeJob{isUpdate: true, record: node}
	}

	// Job done - end of the transaction
	bundle.update <- nodeJob{done: true}
	bundle.job <- nodeJobSignal

	return true
}

// Delete one of the node record specified by host and port
func (bundle *NodeBundle) Delete(host string, port uint64) bool {
	// Lock the bundle for checking record
	bundle.mutex.RLock()

	// Try to find a record
	_, ok := bundle.records[host][port]

	bundle.mutex.RUnlock()

	// if does not exist
	if !ok {
		return false
	}

	// Delete the record
	bundle.update <- nodeJob{
		isDelete: true,
		record:   Node{Host: host, Port: port},
	}

	// Job done - end of the transaction
	bundle.update <- nodeJob{done: true}
	bundle.job <- nodeJobSignal

	return true
}

// DeleteAllByHost - deletes all the nodes records specified by host
func (bundle *NodeBundle) DeleteAllByHost(host string) bool {
	// Lock the bundle for checking record
	bundle.mutex.RLock()
	defer bundle.mutex.RUnlock()

	// Try to find a record
	records, ok := bundle.records[host]

	// if does not exist
	if !ok {
		return false
	}

	// Delete the records
	for port := range records {
		bundle.update <- nodeJob{
			isDelete: true,
			record:   Node{Host: host, Port: port},
		}
	}

	// Job done - end of the transaction
	bundle.update <- nodeJob{done: true}
	bundle.job <- nodeJobSignal

	return true
}

// DeleteAll - deletes all the nodes records
func (bundle *NodeBundle) DeleteAll() {
	// Lock the bundle for the transaction processing
	bundle.mutex.RLock()
	defer bundle.mutex.RUnlock()

	// Delete the records
	for host := range bundle.records {
		for port := range bundle.records[host] {
			bundle.update <- nodeJob{
				isDelete: true,
				record:   Node{Host: host, Port: port},
			}
		}
	}

	// Job done - end of the transaction
	bundle.update <- nodeJob{done: true}
	bundle.job <- nodeJobSignal
}

// InitRing - inits the nodes in the ring ('round-robin') and resets a pointer to the node
func (bundle *NodeBundle) InitRing() {
	nodes, total := bundle.GetAll()
	if bundle.Server.roundRobin && total > 1 {

		// Locks the bundle for the transaction processing
		bundle.mutex.Lock()
		defer bundle.mutex.Unlock()

		bundle.ring = ring.New(len(nodes))
		for _, node := range nodes {
			bundle.ring.Value = node
			bundle.ring = bundle.ring.Next()
		}
	}

}

// CurrentFromRing gets a current Node from the the ring ('round-robin')
func (bundle *NodeBundle) CurrentFromRing() (Node, bool) {
	// Lock the bundle for 'read' operation
	bundle.mutex.RLock()
	defer bundle.mutex.RUnlock()

	node, ok := bundle.ring.Value.(Node)
	return node, ok
}

// TwistRing - sets a pointer to the next node from the ring
func (bundle *NodeBundle) TwistRing() {
	// Lock the bundle for the transaction processing
	bundle.mutex.Lock()
	defer bundle.mutex.Unlock()

	bundle.ring = bundle.ring.Next()
}

// updateRecords is method which does exclusive update/delete of the records
func (bundle *NodeBundle) updateRecords() {

	// Locks the bundle for the transaction processing
	bundle.mutex.Lock()
	defer bundle.mutex.Unlock()

	for {
		update := <-bundle.update

		// If the job is done, unlocks the bundle
		if update.done {
			return
		}

		if update.isDelete {
			queueID := fmt.Sprintf("%s:%d", update.record.Host, update.record.Port)
			stdlog.Println("delete node", update.record.Host, update.record.Port)
			delete(bundle.records[update.record.Host], update.record.Port)
			if len(bundle.records[update.record.Host]) == 0 {
				delete(bundle.records, update.record.Host)
			}
			// removes update channel
			bundle.queues.remove(queueID, bundle.Server.responseTimeout)
		}
		if update.isUpdate {
			queueID := fmt.Sprintf("%s:%d", update.record.Host, update.record.Port)
			stdlog.Println("update node", update.record.Host, update.record.Port)
			// Checks if host does not exist
			if _, ok := bundle.records[update.record.Host]; !ok {
				bundle.records[update.record.Host] = make(map[uint64]Node)
			}
			bundle.records[update.record.Host][update.record.Port] = update.record

			if update.record.Active {
				// Checks the queue, if the queue does not exist,
				// creates a new one and assigns the worker for it
				queue, ok := bundle.queues.check(queueID)
				if !ok {
					if !update.record.Maintenance {
						// creates the worker and assign it to the queue
						go bundle.Server.worker(queue)
					}
				} else {
					if update.record.Maintenance {
						// if the worker is alive
						if getResponse(queue, bundle.Server.responseTimeout) {

							// sends a 'quit' command to the worker
							queue.quit <- struct{}{}

							// gets a response from the worker
							<-queue.response
						}
					} else {
						// if the worker is not alive
						if !getResponse(queue, bundle.Server.responseTimeout) {
							go bundle.Server.worker(queue)
						}
					}
				}
			} else {
				// Removes a channel if it is not active
				// There are removing the worker also
				bundle.queues.remove(queueID, bundle.Server.responseTimeout)
			}
		}
	}
}

// --------------------
// HTTP request methods
// --------------------

// getRecord - gets one of the node record specified by host and port
func (bundle *NodeBundle) getRecord(c *router.Control) {
	c.UseTimer()

	// Try to decode host
	host, ok := decodeString(":host", c)
	if !ok {
		return
	}

	// Try to decode Quantity
	port, ok := decodeNumber(":port", c)
	if !ok {
		return
	}

	// Try to find a record
	record, ok := bundle.Get(host, port)
	if !ok {
		recordNotFound(c)
		return
	}

	result := data{
		"success": true,
		"total":   1,
		"results": []Node{record},
	}
	c.Code(http.StatusOK).Body(result)
}

// getAllRecordsByHost - gets all the nodes records wich contain only specified host
func (bundle *NodeBundle) getAllRecordsByHost(c *router.Control) {
	c.UseTimer()

	// Try to decode host
	host, ok := decodeString(":host", c)
	if !ok {
		return
	}

	nodes, total := bundle.GetAllByHost(host)

	// if records do not exist
	if total == 0 {
		recordNotFound(c)
		return
	}

	result := data{
		"success": true,
		"total":   total,
		"results": nodes,
	}
	c.Code(http.StatusOK).Body(result)
}

// getAllRecords - gets all the nodes records
func (bundle *NodeBundle) getAllRecords(c *router.Control) {
	c.UseTimer()

	// Get all
	nodes, total := bundle.GetAll()

	// if records do not exist
	if total == 0 {
		recordNotFound(c)
		return
	}

	result := data{
		"success": true,
		"total":   total,
		"results": nodes,
	}
	c.Code(http.StatusOK).Body(result)
}

// putRecord updates the node record specified by host and port
func (bundle *NodeBundle) putRecord(c *router.Control) {
	c.UseTimer()

	// Try to decode host
	host, ok := decodeString(":host", c)
	if !ok {
		return
	}

	// Try to decode port
	port, ok := decodeNumber(":port", c)
	if !ok {
		return
	}

	// Locks the bundle for a transaction Node
	bundle.mutex.RLock()
	defer bundle.mutex.RUnlock()

	// Try to find a record
	record, exists := bundle.records[host][port]

	// Try to decode record
	if !decodeRecord(&record, c) {
		return
	}

	// Updates/Creates a decoded record
	if exists {
		if (record.Host != "" && record.Host != host) ||
			(record.Port != 0 && record.Port != port) {
			bundle.update <- nodeJob{
				isDelete: true,
				record:   Node{Host: host, Port: port},
			}
		}

		// Validates Host
		if record.Host == "" {
			couldNotBeEmpty("host", c)
			return
		}

		if !checkAlphaNumeric(record.Host, c) {
			return
		}

		// Validates Port
		if record.Port == 0 {
			couldNotBeZero("port", c)
			return
		}

		c.Code(http.StatusAccepted)
	} else {
		record.Host = host
		record.Port = port

		c.Code(http.StatusCreated)
	}

	// Add the record
	bundle.update <- nodeJob{isUpdate: true, record: record}

	// Job done - end of the transaction
	bundle.update <- nodeJob{done: true}
	bundle.job <- nodeJobSignal

	result := data{
		"success": true,
		"total":   1,
		"results": []Node{record},
	}
	c.Body(result)
}

// putAllRecords updates all the nodes records
func (bundle *NodeBundle) putAllRecords(c *router.Control) {
	c.UseTimer()

	var records []Node
	var updates []Node
	var results []Node

	// Try to decode records
	buffer, ok := preDecodeRecords(&records, c)
	if !ok {
		return
	}

	// Locks the bundle for the transaction processing
	bundle.mutex.RLock()
	defer bundle.mutex.RUnlock()

	for _, record := range records {
		if record.Host == "" || record.Port == 0 {
			couldNotBeEmpty("host/port", c)
			return
		}

		if !checkAlphaNumeric(record.Host, c) {
			return
		}

		// Try to find a record
		update, exists := bundle.records[record.Host][record.Port]
		if exists {
			updates = append(updates, update)
		}
	}

	// Try to decode records
	if !postDecodeRecords(buffer, &updates, c) {
		return
	}

	for _, update := range updates {
		// Add record
		bundle.update <- nodeJob{isUpdate: true, record: update}
		results = append(results, update)
	}

	// Job is done - end of the transaction
	bundle.update <- nodeJob{done: true}
	bundle.job <- nodeJobSignal

	result := data{
		"success": true,
		"total":   len(results),
		"results": results,
	}
	c.Code(http.StatusAccepted).Body(result)
}

// deleteRecord deletes one of the node record specified by host and port
func (bundle *NodeBundle) deleteRecord(c *router.Control) {
	c.UseTimer()

	// Try to decode host
	host, ok := decodeString(":host", c)
	if !ok {
		return
	}

	// Try to decode port
	port, ok := decodeNumber(":port", c)
	if !ok {
		return
	}

	if !bundle.Delete(host, port) {
		recordNotFound(c)
		return
	}

	c.Code(http.StatusOK).Body(data{"success": true})
}

// deleteAllRecordsByHost deletes all the nodes records wich contain only specified host
func (bundle *NodeBundle) deleteAllRecordsByHost(c *router.Control) {
	c.UseTimer()

	// Try to decode host
	host, ok := decodeString(":host", c)
	if !ok {
		return
	}

	if !bundle.DeleteAllByHost(host) {
		recordNotFound(c)
		return
	}

	c.Code(http.StatusOK).Body(data{"success": true})
}

// deleteAllRecords deletes all the nodes records
func (bundle *NodeBundle) deleteAllRecords(c *router.Control) {
	c.UseTimer()

	bundle.DeleteAll()

	c.Code(http.StatusOK).Body(data{"success": true})
}
