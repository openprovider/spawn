package spawn

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"
)

func TestNodes(t *testing.T) {
	var nodes []Node

	// set timeout 1 second for testing
	var timeout time.Duration = 1

	// create new server
	server, err := NewServer("test")
	server.byPriority = true
	server.roundRobin = true
	server.Nodes.update = make(chan nodeJob, MaxJobs)
	server.responseTimeout = timeout

	// start server worker, for the nodes testing
	go server.manage()

	// load failed nodes with incorect host's names and port's values
	err = loadFixtures("fixtures/failed_nodes.json", &nodes)
	test(t, err == nil, "Expected loading of failed fixtures, got", err)

	// Try to load failed nodes data
	for _, node := range nodes {
		test(t, !server.Nodes.Set(&node),
			"Expected the fixtures have not been loaded, but it has", node)
	}

	// loads correct nodes data from a file
	err = loadFixtures("fixtures/nodes.json", &nodes)
	test(t, err == nil, "Expected loading of fixtures, got", err)

	// sort the nodes by priority
	sort.Sort(byPriority(nodes))

	// Save correct nodes priority and updates server nodes
	var expectedPriority = make([]int, len(nodes))
	for index, node := range nodes {
		expectedPriority[index] = node.Priority
		test(t, server.Nodes.Set(&node), "Error load fixture:", node)
	}
	test(t, len(expectedPriority) == len(nodes),
		"Expected count of priority", len(expectedPriority), "got", len(nodes))

	// Wait of response after all will be updated
	server.job <- responseSignal
	<-server.response

	// Compare loaded nodes and check workers
	for _, node := range nodes {
		loadedNode, ok := server.Nodes.Get(node.Host, node.Port)
		test(t, ok, "Error load fixture:", node)
		test(t, loadedNode == node,
			"Loaded node has incorrect values, expected", node, "got", loadedNode)
		id := fmt.Sprintf("%s:%d", loadedNode.Host, loadedNode.Port)
		q, ok := server.queue.check(id)
		if !loadedNode.Active {
			test(t, !ok, "Expected the queue does not exist, got", q)
		} else {
			test(t, ok, "Expected the queue exists, got it does not exist", node)
		}
		if loadedNode.Active && !loadedNode.Maintenance {
			test(t, getResponse(q, timeout), "Expected the worker was loaded, got timeout")
		} else {
			test(t, !getResponse(q, timeout), "Expected the worker is inactive, got is active")
		}
	}

	// load all nodes again
	loadedNodes, total := server.Nodes.GetAll()
	for index, node := range loadedNodes {
		test(t, node.Priority == expectedPriority[index],
			"Expected priority is", expectedPriority[index], "got", node.Priority)

		// get current node from 'round-robin' ring
		ringNode, ok := server.Nodes.CurrentFromRing()
		test(t, ok, "Expexted loading current node from 'round-robin' ring, got nothing")
		test(t, ringNode.Priority == expectedPriority[index],
			"Expected priority of the node from 'round-robin' ring is",
			expectedPriority[index], "got", ringNode.Priority)

		// set pointer to the next node from the ring
		server.Nodes.TwistRing()
	}
	test(t, server.Nodes.DeleteAllByHost(nodes[0].Host),
		"Expected the nodes have been deleted got have not")
	test(t, total == len(nodes), "Expected count of nodes", len(nodes), "got", total)
	test(t, server.Nodes.SetAll(nodes),
		"Expected the nodes will updated successfully, got that was not")

	// load the nodes sorted by Host
	loadedNodes, total = server.Nodes.GetAllByHost(nodes[0].Host)
	test(t, total == len(nodes), "Expected count of nodes", len(nodes), "got", total)

	// delete the nodes by host and port
	for _, node := range loadedNodes {
		test(t, server.Nodes.Delete(node.Host, node.Port),
			"Expected the node has been deleted got has not")
	}

	// Wait of response after all will be updated
	server.job <- responseSignal
	<-server.response

	// check the queues of the nodes, must be absent
	for _, node := range loadedNodes {
		id := fmt.Sprintf("%s:%d", node.Host, node.Port)
		q, ok := server.queue.check(id)
		test(t, !ok, "Expected the queue does not exist, got", q)
	}
}

// loadFixtures - loads fixtures
func loadFixtures(path string, nodes *[]Node) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := json.NewDecoder(bufio.NewReader(file)).Decode(&nodes); err != nil {
		return err
	}

	return nil
}
