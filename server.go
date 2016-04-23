// Copyright 2016 Openprovider Authors. All rights reserved.
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
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/openprovider/spawn/auth"
	"github.com/takama/router"
)

const (

	// VERSION - current version of the service
	VERSION = "0.3.4"

	// DATE - revision date of the service
	DATE = "2016-04-23T09:50:17Z"

	// MaxSignals - maximum count of update signals
	MaxSignals = 1000

	// MaxJobs - maximum count of update jobs for every bundle
	MaxJobs = 100000

	// DefaultTimeout is a timeout for the worker's response
	DefaultTimeout time.Duration = 10

	// HTTP methods, which should be queued
	protocolHTTP = "http"
	methodGET    = "GET"
	methodPOST   = "POST"
	methodPUT    = "PUT"
	methodDELETE = "DELETE"

	// Job signals
	responseSignal = iota
	nodeJobSignal
)

// simplest logger, which initialized during starts of the application
var (
	stdlog = log.New(os.Stdout, "[CORE]: ", log.LstdFlags)
	errlog = log.New(os.Stderr, "[CORE:ERROR]: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// Server Record
type Server struct {

	// Server name/description
	Name string

	// Embeded router
	*router.Router

	// Node Bundle contains the Node records
	Nodes *NodeBundle
	// contains filtered or unexported fields

	// Entry Bundle contains the entry methods
	entry *entryBundle

	// Queue Bundle contains the queue records
	queues *queueBundle

	// round robin mode
	roundRobin bool

	// nodes will queried according to priority
	byPriority bool

	// nodes health check
	check HealthCheck

	// Node connection transport
	transport http.RoundTripper

	// responseTimeout is a timeout for worker's response
	responseTimeout time.Duration

	// job signal channel
	job chan int

	// Response signal channel
	response chan struct{}

	// quit signal channel
	quit chan struct{}
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

	// Init the Server
	server := &Server{
		Name:            name,
		Router:          router.New(),
		transport:       http.DefaultTransport,
		responseTimeout: DefaultTimeout,
		job:             make(chan int, MaxSignals),
		response:        make(chan struct{}, MaxSignals),
		quit:            make(chan struct{}, 1),
	}

	server.Router.PanicHandler = func(c *router.Control) {
		c.Code(http.StatusInternalServerError).Body(c.Request)
	}
	server.Router.NotFound = notFound
	server.Router.Logger = logger
	server.Router.CustomHandler = server.baseHandler

	// Create and init nodes bundle
	server.Nodes = &NodeBundle{
		Server:  server,
		records: make(map[string]map[uint64]Node),
	}

	// Create and init queues bundle
	server.queues = &queueBundle{records: make(map[string]*queue)}

	return server, nil
}

// Run the server, init the handlers, init the specified modes.
// If transport http.RoundTripper is not defined will be used default transport.
// http.RoundTripper contains callback function which handle
// all incoming requests and get responses/errors
func (server *Server) Run(
	hostPort, apiHostPort string,
	transport http.RoundTripper,
	nodes []Node,
	roundRobin, byPriority bool,
	check HealthCheck,
	authService auth.Auth,
) (status string, err error) {

	// if used round-robin mode
	if roundRobin {
		stdlog.Println(server.Name, "server is using 'round-robin' mode")
		server.roundRobin = roundRobin
	}

	// if used by-priority mode
	if byPriority {
		stdlog.Println("The nodes are operating according to priority")
		server.byPriority = byPriority
	}

	// Init the Nodes update channel
	server.Nodes.update = make(chan nodeJob, MaxJobs)

	// Starts the worker which manage server's jobs
	go server.jobListener()

	// Init the Nodes settings
	if !server.Nodes.SetAll(nodes) {
		status = server.Name + " is not loaded"
		err = errors.New("The config parameters for the nodes have incorrect values")
		return
	}

	// Init a health check settings
	server.check = check

	// Init auth service
	server.entry = &entryBundle{
		Auth: authService,
	}

	server.setupRoutes()

	go server.Listen(apiHostPort)
	go func() {
		p := &proxy{transport: server}
		if transport != nil {
			p.transport = transport
		}
		if err := http.ListenAndServe(hostPort, p); err != nil {
			errlog.Fatal(err)
		}
	}()

	status = server.Name + " is loaded successfully"

	return
}

// Shutdown closes the server graceful
func (server *Server) Shutdown() (status string, err error) {

	// Set timer to wait one minute
	timeout := time.NewTimer(time.Minute)

	// sweeps all responses if exist
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

	status = server.Name + " server connections are closed"
	select {

	// Exit by timeout if jobs have not done
	case <-timeout.C:
		err = errors.New("timeout")
		return
	// Exit after doing all jobs
	case <-server.response:
		return
	}
}

func (server *Server) setupRoutes() {
	// The info handler returns a system status of the application
	server.GET("/info", infoHandler)

	// Lists methods, which display how to use API
	server.GET("/list", displayAllMethods)
	server.GET("/list/nodes", displayAllNodeMethods)
	server.GET("/list/nodes/get", displayGetNodeMethods)
	server.GET("/list/nodes/set", displaySetNodeMethods)
	server.GET("/list/nodes/delete", displayDeleteNodeMethods)

	// Entry methods
	server.POST("/login", server.entry.login)
	server.GET("/login/:token", server.entry.info)
	server.DELETE("/logout/:token", server.entry.logout)
	server.OPTIONS("/login", optionsHandler)
	server.OPTIONS("/login/:token", optionsHandler)
	server.OPTIONS("/logout/:token", optionsHandler)

	// Init API methods for the Nodes
	server.GET("/nodes/:host/:port", server.Nodes.getRecord)
	server.GET("/nodes/:host", server.Nodes.getAllRecordsByHost)
	server.GET("/nodes", server.Nodes.getAllRecords)
	server.PUT("/nodes/:host/:port", server.Nodes.putRecord)
	server.PUT("/nodes", server.Nodes.putAllRecords)
	server.DELETE("/nodes/:host/:port", server.Nodes.deleteRecord)
	server.DELETE("/nodes/:host", server.Nodes.deleteAllRecordsByHost)
	server.DELETE("/nodes", server.Nodes.deleteAllRecords)
	server.OPTIONS("/nodes", optionsHandler)
	server.OPTIONS("/nodes/:host", optionsHandler)
	server.OPTIONS("/nodes/:host/:port", optionsHandler)
}

// jobListener is routine which listen job signals and activate job controller
func (server *Server) jobListener() {
	defer func() {
		if recovery := recover(); recovery != nil {
			errlog.Println("Recovered in job listener routine", recovery)
			// Recover routine
			go server.jobListener()
		} else {
			stdlog.Println("Listener routine is stopped")
			server.response <- struct{}{}
		}
	}()
	for {
		select {
		case job := <-server.job:
			server.jobController(job)
			continue
		default:
		}
		select {
		case job := <-server.job:
			server.jobController(job)
			continue
		case <-server.quit:
			server.entry.Close()
			return
		}
	}
}

// jobController starts jobs depended from signal
func (server *Server) jobController(signal int) {
	defer func() {
		if recovery := recover(); recovery != nil {
			errlog.Println("Recovered in job controller routine", recovery)
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

// RoundTrip manages all requests/responses
func (server *Server) RoundTrip(request *http.Request) (*http.Response, error) {

	// Add "X-Forwarded-For" to repost remote host IP
	if request.Header.Get("X-Forwarded-For") == "" {
		request.Header.Add("X-Forwarded-For", request.RemoteAddr)
	}

	// Use HTTP scheme
	request.URL.Scheme = protocolHTTP

	// If requests could not be queued, get result immediately
	if request.Method != methodPOST &&
		request.Method != methodPUT &&
		request.Method != methodDELETE {

		return server.processReceive(request)
	}

	return server.processUpdate(request)
}

// calls 'GET' and others requests to the node using defined mode
func (server *Server) processReceive(request *http.Request) (*http.Response, error) {
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
					response, err := server.transport.RoundTrip(request)
					if err == nil {
						// If response is sucess, return
						return response, nil
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
							return response, nil
						}
						errlog.Println(err)
					}
				}
			}
		}
	}

	return nil, errors.New("Warning: no one of the nodes is active")
}

// call 'PUT', 'POST', 'DELETE' request to the node
func (server *Server) processUpdate(request *http.Request) (*http.Response, error) {
	// grab update request
	proxyRequestData, err := httputil.DumpRequest(request, true)
	if err != nil {

		// if unsuccessful, return error
		return nil, err
	}
	var host string
	var response *http.Response
	if nodes, total := server.Nodes.GetAll(); total > 0 {
		answer := make(chan *http.Response, total)
		done := make(chan struct{}, total)
		for _, node := range nodes {
			if node.Active {

				host = fmt.Sprintf("%s:%d", node.Host, node.Port)

				// create new queue job
				job := &queueJob{
					done:   done,
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
				return response, nil
			case <-timeout.C:
				return response, errors.New("timeout")
			}
		}
	}
	return response, errors.New("The nodes are not defined")
}

// worker receives a data from the queue and send it to the node
func (server *Server) worker(q *queue) {
	defer func() {
		if recovery := recover(); recovery != nil {
			errlog.Println("Recovered in worker routine", recovery)
			// the worker recovers again
			go server.worker(q)
		} else {
			q.response <- struct{}{}
			stdlog.Println("Worker is closed for", q.id)
		}
	}()
	stdlog.Println("Worker is started for", q.id)
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
	if response, err := server.dispatchRequest(q.id, data); err != nil {

		// Job does not done
		errlog.Println(err)

	} else {

		// job done
		if len(job.done) == 0 {
			// send first response and done signal
			job.done <- struct{}{}
			job.answer <- response
		} else {
			// just close connection
			response.Body.Close()
		}
	}
}

// checks the node
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

// Reproduces request to specified node and capture response
func (server *Server) dispatchRequest(host string, data []byte) (*http.Response, error) {
	reader := bufio.NewReader(bytes.NewBuffer(data))
	request, err := http.ReadRequest(reader)
	if err != nil {
		return nil, err
	}
	request.Body = ioutil.NopCloser(reader)
	request.URL.Scheme = protocolHTTP
	request.URL.Host = host

	response, err := server.transport.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (server *Server) baseHandler(handle router.Handle) router.Handle {
	return func(c *router.Control) {
		if c.Get("pretty") != "true" {
			c.CompactJSON(true)
		}
		if origin := c.Request.Header.Get("Origin"); origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "false")
		}
		if method := c.Request.Header.Get("Access-Control-Request-Method"); method != "" {
			allowedMethods := server.Router.AllowedMethods(c.Request.URL.Path)
			c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
		}
		if headers := c.Request.Header.Get("Access-Control-Request-Headers"); headers != "" {
			c.Writer.Header().Set("Access-Control-Allow-Headers", "content-type")
		}
		handle(c)
	}
}
