package spawn

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/openprovider/spawn/auth"
)

type testAnswer struct {
	Node   string `json:"node"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Data   []byte `json:"data"`
}

type testContent struct {
	Method    string `json:"method"`
	Iteration int    `json:"iteration"`
}

type testUpdatesStats struct {
	mutex    sync.RWMutex
	sent     int
	received map[string]int
}

var stats testUpdatesStats

type testConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`

	API struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"api"`

	Nodes []Node `json:"nodes"`

	AuthEngine auth.AuthConfig `json:"auth"`

	FailedNodeID string `json:"failedNodeID"`
}

// LoadConfigFile - loads congig file into config record
func (config *testConfig) loadTestConfig(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
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

func TestServer(t *testing.T) {

	stats.received = make(map[string]int)

	// create new server
	server, err := NewServer("Test-server")
	test(t, err == nil, "Expected create a new server, got", err)
	server.responseTimeout = 1

	// loads test config
	config := &testConfig{}
	err = config.loadTestConfig("fixtures/config.json")
	test(t, err == nil, "Expected loading of the test hosts, got", err)

	// run the server in testing mode
	appHost := fmt.Sprintf("%s:%d", config.Host, config.Port)
	apiHost := fmt.Sprintf("%s:%d", config.API.Host, config.API.Port)

	// Initialize auth service
	authService, err := auth.NewAuth(&config.AuthEngine)
	test(t, err == nil, "Expected new auth service, got", authService, err)
	status, err := server.Run(
		appHost, apiHost, nil, config.Nodes,
		true, true, HealthCheck{Seconds: 1}, authService,
	)
	test(t, err == nil, "Expected run the server, got", status, err)

	// the server must provide /info and /list methods
	// Test /info method
	testPath := "/info"
	url := "http://" + apiHost + testPath
	response, err := http.Get(url)
	test(t, err == nil, "Expected to get /info method, got", err)
	test(t, response.StatusCode == http.StatusOK,
		"Expected to get /info method with ok status, got", response.StatusCode)

	// Test /list method
	testPath = "/list"
	url = "http://" + apiHost + testPath
	response, err = http.Get(url)
	test(t, err == nil, "Expected to get /list method, got", err)
	test(t, response.StatusCode == http.StatusOK,
		"Expected to get /list method with ok status, got", response.StatusCode)

	// activate nodes (listen and serve)
	for _, node := range config.Nodes {
		id := fmt.Sprintf("%s:%d", node.Host, node.Port)
		if config.FailedNodeID != id {
			go func() {
				err := http.ListenAndServe(id, &testProxy{node: id})
				test(t, err == nil, "Expected of activate nodes without errors, got", err)
			}()
		}
	}

	// Wait of response after the nodes will be updated
	server.job <- responseSignal
	<-server.response

	// Test GET (round-robin is on, by-priority is on)
	testPath = "/test"
	url = "http://" + appHost + testPath

	sort.Sort(byPriority(config.Nodes))

	client := http.DefaultClient
	content := new(testContent)

	// do 100 times of GET request to different host (in order according to priority)
	for i := 1; i <= 100; i++ {
		// for every node
		for _, node := range config.Nodes {
			id := fmt.Sprintf("%s:%d", node.Host, node.Port)
			if node.Active && !node.Maintenance && config.FailedNodeID != id {

				content.Method = "GET"
				content.Iteration = i
				bytesContent, err := json.Marshal(content)
				test(t, err == nil, "Expected to marshal content, got", err)

				requestGET, err := http.NewRequest("GET", url, bytes.NewReader(bytesContent))
				test(t, err == nil, "Expected to create new GET request, got", err)

				// just GET always the same URL
				response, err := client.Do(requestGET)
				test(t, err == nil, "Expected GET from app host, got", err)
				test(t, response.StatusCode == http.StatusOK,
					"Expected status ok for GET app host, got", response.StatusCode)

				body, err := ioutil.ReadAll(response.Body)
				response.Body.Close()
				test(t, err == nil, "Expected read body response, got", err)

				// got answer
				answer := new(testAnswer)
				test(t, json.Unmarshal(body, answer) == nil,
					"Expected unmarshall of the body, got", err)

				// test the node id will match with sequence of the hosts sorted by priority,
				// will having active status without maintenance
				test(t, answer.Node == id, "Expected node id", id, ", got", answer.Node)

				// the URL must match
				test(t, answer.Path == testPath,
					"Expected root url:", testPath, ", got", answer.Path)
				// the method must match
				test(t, response.Request.Method == answer.Method,
					"Expected method:", response.Request.Method, "got", answer.Method)
				// compare that content the same
				compareContent := new(testContent)
				err = json.Unmarshal(answer.Data, compareContent)
				test(t, compareContent.Method == content.Method,
					"Expected recieved method should be equal, got", compareContent.Method)
				test(t, compareContent.Iteration == content.Iteration,
					"Expected recieved iteration should be equal, got", compareContent.Iteration)
			}
		}
	}

	// do 100 times of update request to different host (in order according to priority)
	for i := 1; i <= 100; i++ {
		// for every node
		for _, operation := range []string{"POST", "PUT", "DELETE"} {

			content.Method = operation
			content.Iteration = i

			bytesContent, err := json.Marshal(content)
			test(t, err == nil, "Expected to marshal content, got", err)

			request, err := http.NewRequest(operation, url, bytes.NewReader(bytesContent))
			test(t, err == nil, "Expected to create new update request, got", err)

			response, err := client.Do(request)
			test(t, err == nil, "Expected update data to app host, got", err)
			test(t, response.StatusCode == http.StatusOK,
				"Expected status ok for update data to app host, got", response.StatusCode)
			stats.mutex.Lock()
			stats.sent++
			stats.mutex.Unlock()
			body, err := ioutil.ReadAll(response.Body)
			response.Body.Close()
			test(t, err == nil, "Expected read body response, got", err)

			// got answer
			answer := new(testAnswer)
			test(t, json.Unmarshal(body, answer) == nil,
				"Expected unmarshall of the body, got", err)

			// the URL must match
			test(t, answer.Path == testPath,
				"Expected root url:", testPath, ", got", answer.Path)
			// the method must match
			test(t, operation == answer.Method,
				"Expected method:", operation, "got", answer.Method)
			// compare that content the same
			compareContent := new(testContent)
			err = json.Unmarshal(answer.Data, compareContent)
			test(t, compareContent.Method == content.Method,
				"Expected recieved method should be equal, got", compareContent.Method)
			test(t, compareContent.Iteration == content.Iteration,
				"Expected recieved iteration should be equal, got", compareContent.Iteration)
		}
	}

	// waiting for all updates were sent to nodes which are active,
	// not in maintenance mode and not failed
	for _, node := range config.Nodes {
		id := fmt.Sprintf("%s:%d", node.Host, node.Port)
		if node.Active && !node.Maintenance && config.FailedNodeID != id {
			q, ok := server.queues.check(id)
			test(t, ok, "Expected queue should be exist, got it does not exist")
			getResponse(q, 1)
			stats.mutex.RLock()
			test(t, stats.received[id] == 300,
				"Expected count of updates for", id, "- 300, got", stats.received[id])
			stats.mutex.RUnlock()
		}
	}
	// waiting for all updates were sent to nodes which are active
	// and will be switched off maintenance mode
	for _, node := range config.Nodes {
		id := fmt.Sprintf("%s:%d", node.Host, node.Port)
		if node.Active && node.Maintenance {
			q, ok := server.queues.check(id)
			test(t, ok, "Expected queue should be exist, got it does not exist")

			// switch off maintenance mode
			node.Maintenance = false
			test(t, server.Nodes.Set(&node),
				"Expected change maintenance mode for", id, ", got error")
			getResponse(q, 3)
			test(t, stats.received[id] == 300,
				"Expected count of updates for", id, "- 300, got", stats.received[id])
		}
	}

	// waiting for all updates were sent to nodes which was failed and become to active
	for _, node := range config.Nodes {
		id := fmt.Sprintf("%s:%d", node.Host, node.Port)
		if node.Active && !node.Maintenance && config.FailedNodeID == id {
			q, ok := server.queues.check(id)
			test(t, ok, "Expected queue should be exist, got it does not exist")

			// restore failed node
			go func() {
				err := http.ListenAndServe(config.FailedNodeID, &testProxy{node: config.FailedNodeID})
				test(t, err == nil, "Expected of activate nodes without errors, got", err)
			}()
			// waiting for node become alive
			time.Sleep(time.Second)

			// waiting for all updates came
			getResponse(q, 3)
			stats.mutex.RLock()
			test(t, stats.received[id] == 300,
				"Expected count of updates for", id, "- 300, got", stats.received[id])
			stats.mutex.RUnlock()
		}
	}

	// shutdown the server
	status, err = server.Shutdown()
	test(t, err == nil, "Expected shutdown the server, got", status, err)
}

type testProxy struct {
	node string
}

func (tp *testProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		errlog.Println("Read request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	response := data{
		"node":   tp.node,
		"method": req.Method,
		"path":   req.URL.RequestURI(),
		"data":   body,
	}
	content, err := json.Marshal(response)
	if err != nil {
		errlog.Println("Read request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.Method == "PUT" || req.Method == "POST" || req.Method == "DELETE" {
		stats.mutex.Lock()
		stats.received[tp.node]++
		stats.mutex.Unlock()
	}
	w.Header().Add("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}
