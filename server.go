// Copyright 2015 Igor Dolzhikov. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package spawn

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/takama/router"
)

const (

	// VERSION - current version of the service
	VERSION = "0.1.4"

	// DATE - revision date of the service
	DATE = "2015-03-24T22:17:17Z"

	// MaxSignals - maximum count of update signals
	MaxSignals = 1000

	// MaxJobs - maximum count of update jobs for every bundle
	MaxJobs = 100000

	// DefaultTimeout is a timeout for the worker's response
	DefaultTimeout time.Duration = 10

	// HTTP methods, which should be queued
	protocolHTTP = "http"
	methodPOST   = "POST"
	methodPUT    = "PUT"
	methodDELETE = "DELETE"

	// Job signals
	responseSignal = iota
	nodeJobSignal
)

// simplest logger, which initialized during starts of the application
var (
	stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lshortfile)
)

// Server Record
type Server struct {

	// Server name/description
	Name string

	// Node Bundle contains the Node records
	Nodes *NodeBundle
	// contains filtered or unexported fields

	// Embeded router
	*router.Router

	// Response signal channel
	response chan struct{}

	// responseTimeout is a timeout for worker's response
	responseTimeout time.Duration

	// job signal channel
	job chan int

	// quit signal channel
	quit chan struct{}

	// round robin mode
	roundRobin bool

	// nodes will queried according to priority
	byPriority bool

	// nodes health check
	check HealthCheck

	// Queue Bundle contains the queue records
	queues *queueBundle
}

// HealthCheck contains parameters which used for checking node
type HealthCheck struct {

	// health check time of the node in seconds
	Seconds time.Duration `json:"seconds"`

	// url which will be checked
	URL string `json:"url"`

	// regexp pattern for extended check analyze
	Pattern string `json:"regexp"`
}

// NewServer creates a new server which contains the nodes/queues
func NewServer(name string) (*Server, error) {

	// Init the router
	r := router.New()
	r.PanicHandler = panicHandler
	r.NotFound = notFound
	r.Logger = logger
	r.CustomHandler = baseHandler

	// Init the Server
	server := &Server{
		Name:            name,
		Router:          r,
		responseTimeout: DefaultTimeout,
		response:        make(chan struct{}),
		job:             make(chan int, MaxSignals),
		quit:            make(chan struct{}),
	}

	// Create and init nodes bundle
	server.Nodes = &NodeBundle{
		Server:  server,
		records: make(map[string]map[uint64]Node),
	}

	// Create and init queues bundle
	server.queues = &queueBundle{records: make(map[string]*queue)}

	return server, nil
}

// Run the server, init the handlers, init the specified modes
// If handler RequestHandler is not defined used default handler
// with standard handling of the requests/responses
func (server *Server) Run(
	hostPort, apiHostPort string,
	handler RequestHandler,
	nodes []Node,
	roundRobin, byPriority bool,
	check HealthCheck,
) (status string, err error) {

	// if used round-robin mode
	if roundRobin {
		stdlog.Println(server.Name, "will used 'round-robin' mode")
		server.roundRobin = roundRobin
	}

	// if used by-priority mode
	if byPriority {
		stdlog.Println("Nodes will queried according to priority")
		server.byPriority = byPriority
	}

	// Init the Nodes update channel
	server.Nodes.update = make(chan nodeJob, MaxJobs)

	// Starts the worker which manage server's jobs
	go server.manage()

	// Init the Nodes settings
	if !server.Nodes.SetAll(nodes) {
		status = server.Name + " is not loaded"
		err = errors.New("The nodes settings in config have incorrect values")
		return
	}

	// Init a health check settings
	server.check = check

	// The info handler returns a system status of the application
	server.GET("/info", infoHandler)

	// Lists methods, which display how to use API
	server.GET("/list", displayAllMethods)
	server.GET("/list/nodes", displayNodeMethods)

	// Init API methods for the Nodes
	server.GET("/nodes/:host/:port", server.Nodes.getRecord)
	server.GET("/nodes/:host", server.Nodes.getAllRecordsByHost)
	server.GET("/nodes", server.Nodes.getAllRecords)
	server.PUT("/nodes/:host/:port", server.Nodes.putRecord)
	server.PUT("/nodes", server.Nodes.putAllRecords)
	server.DELETE("/nodes/:host/:port", server.Nodes.deleteRecord)
	server.DELETE("/nodes/:host", server.Nodes.deleteAllRecordsByHost)
	server.DELETE("/nodes", server.Nodes.deleteAllRecords)

	go server.Listen(apiHostPort)
	go func() {
		p := new(proxy)
		if handler != nil {
			p.handler = handler
		} else {
			p.handler = server.proxyHandler
		}
		if err := http.ListenAndServe(hostPort, p); err != nil {
			errlog.Fatal(err)
		}
	}()

	status = server.Name + " loaded successfully"

	return
}

// Shutdown closes the server graceful
func (server *Server) Shutdown() (status string, err error) {

	// Set timer to wait one minute
	timeout := time.NewTimer(time.Second * 30)

	// a unwanted response sweeps if exist
	for {
		select {
		case <-server.response:
			continue
		default:
		}
		break
	}

	// sends a 'quit' signal
	server.quit <- struct{}{}

	status = server.Name + " connections closed"
	select {

	// Exit by timeout if jobs did not done
	case <-timeout.C:
		err = errors.New("timeout")
		return
	// Exit after all jobs done
	case <-server.response:
		return
	}
}

// Manage routine which manage all jobs
func (server *Server) manage() {
	defer func() {
		if recovery := recover(); recovery != nil {
			errlog.Println("Recovered in Manage routine", recovery)
			// Recover routine
			go server.manage()
		} else {
			stdlog.Println("Manage routine stoped")
			server.response <- struct{}{}
		}
	}()
	for {
		select {
		case job := <-server.job:
			server.doJob(job)
			continue
		default:
		}
		select {
		case job := <-server.job:
			server.doJob(job)
			continue
		case <-server.quit:
			return
		}
	}
}

// Do updates depended by the signal
func (server *Server) doJob(signal int) {
	defer func() {
		if recovery := recover(); recovery != nil {
			errlog.Println("Recovered in do job routine", recovery)
		}
	}()
	switch signal {
	case responseSignal:
		server.response <- struct{}{}
	case nodeJobSignal:
		server.Nodes.updateRecords()
		server.Nodes.InitRing()
	}
}

// proxyHandler manages all requests/responses
func (server *Server) proxyHandler(request *http.Request) *http.Response {

	// Add "X-Forwarded-For" to repost remote host IP
	if remoteHost, _, err := net.SplitHostPort(request.RemoteAddr); err == nil {
		request.Header.Add("X-Forwarded-For", remoteHost)
	}
	// Use HTTP scheme
	request.URL.Scheme = protocolHTTP

	// If requests should not be queued, get result immediately
	if request.Method != methodPOST &&
		request.Method != methodPUT &&
		request.Method != methodDELETE {

		return server.processGET(request)
	}

	return server.processUpdate(request)
}

// call 'GET' request to the node using defined mode
func (server *Server) processGET(request *http.Request) *http.Response {
	if server.roundRobin {

		// Use round robin to get data from the host
		for count := 0; count < server.Nodes.ring.Len(); count++ {
			if node, ok := server.Nodes.CurrentFromRing(); ok &&
				node.Active && !node.Maintenance {

				// The host is active and is not in maintenance
				request.URL.Host = fmt.Sprintf("%s:%d", node.Host, node.Port)

				// Prepare next host
				server.Nodes.TwistRing()

				if server.checkNode(request.URL.Host) {
					response, err := http.DefaultTransport.RoundTrip(request)
					if err == nil {
						// If response is sucess, return
						return response
					}
					errlog.Println(err)
				}
			} else {

				// Use next host if not active or maintenance mode
				server.Nodes.TwistRing()
			}
		}
	} else {

		// If is not round robin mode, use first registered host
		if nodes, total := server.Nodes.GetAll(); total > 0 {
			if server.byPriority {
				sort.Sort(byPriority(nodes))
			}
			for _, node := range nodes {
				if node.Active && !node.Maintenance {

					// The host is active and is not in maintenance
					request.URL.Host = fmt.Sprintf("%s:%d", node.Host, node.Port)
					if server.checkNode(request.URL.Host) {
						response, err := http.DefaultTransport.RoundTrip(request)
						if err == nil {
							// If response is sucess, return
							return response
						}
						errlog.Println(err)
					}
				}
			}
		}
	}

	stdlog.Println("Warning: no one of the nodes is active")

	return nil
}

// call 'PUT', 'POST', 'DELETE' request to the node
func (server *Server) processUpdate(request *http.Request) *http.Response {
	// grab update request
	proxyRequestData, err := httputil.DumpRequest(request, true)
	if err != nil {

		// if unsuccessful, return nil response
		errlog.Println(err)
		return nil
	}
	var host string
	var response *http.Response
	if nodes, total := server.Nodes.GetAll(); total > 0 {
		answer := make(chan *http.Response, total)
		for _, node := range nodes {
			if node.Active {

				host = fmt.Sprintf("%s:%d", node.Host, node.Port)

				// create new queue job
				job := &queueJob{
					query:  make(chan []byte, 1),
					answer: answer,
				}
				job.query <- proxyRequestData

				queue, _ := server.queues.check(host)
				queue.jobs <- job
				queue.task <- doJobTask
			}
		}
		timeout := time.NewTimer(time.Second * server.responseTimeout)
		for {
			select {
			case response = <-answer:
				return response
			case <-timeout.C:
				return response
			}
		}
	}
	return response
}

// worker receive a data from the queue and send it to the node
func (server *Server) worker(q *queue) {
	defer func() {
		if recovery := recover(); recovery != nil {
			errlog.Println("Recovered in worker routine", recovery)
			// the worker recovers again
			go server.worker(q)
		} else {
			q.response <- struct{}{}
			stdlog.Println("Worker closed for", q.id)
		}
	}()
	stdlog.Println("Worker started for", q.id)
	for {
		select {
		case task := <-q.task:
			switch task {
			case doJobTask:
				server.doUpdate(q)
			}
			continue
		default:
		}
		select {
		case task := <-q.task:
			switch task {
			case doJobTask:
				server.doUpdate(q)
			}
			continue
		case <-q.quit:
			return
		case <-q.ask:
			q.response <- struct{}{}
		}
	}
}

func (server *Server) doUpdate(q *queue) {
	// check the node
	for {
		if server.checkNode(q.id) {
			break
		}
		stdlog.Println("Node", q.id, "does not ready for updates")
		stdlog.Println("try again in", server.check.Seconds, "seconds")
		timeout := time.NewTimer(time.Second * server.check.Seconds)
		select {
		//  Repeat by timeout
		case <-timeout.C:
			continue
		case <-q.quit:
			q.task <- doJobTask
			return
		case <-q.ask:
			q.response <- struct{}{}
		}
	}
	// if the node is alive, post data
	job := <-q.jobs
	data := <-job.query
	if response, err := dispatchRequest(q.id, data); err != nil {

		// Job does not done
		errlog.Println(err)

	} else {

		// job done
		job.answer <- response
		job.done = true
	}
}

// check the node
func (server *Server) checkNode(host string) bool {
	response, err := http.Get(protocolHTTP + "://" + host + server.check.URL)
	if err != nil {
		return false
	}

	defer response.Body.Close()
	// if pattern does not exist, should be true
	if server.check.Pattern == "" {
		return true
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return false
	}
	// check of regexp pattern
	valid := regexp.MustCompile(server.check.Pattern)
	return valid.MatchString(string(data))
}

// Reproduce request to specified node and capture response
func dispatchRequest(host string, data []byte) (*http.Response, error) {
	reader := bufio.NewReader(bytes.NewBuffer(data))
	request, err := http.ReadRequest(reader)
	if err != nil {
		return nil, err
	}
	request.Body = ioutil.NopCloser(reader)
	request.URL.Scheme = protocolHTTP
	request.URL.Host = host

	response, err := http.DefaultTransport.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}
