// Copyright 2016 Openprovider Authors. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package spawn

import (
	"net/http"
	"sync"
	"time"
)

const (
	cmdQueueCapacity = 100

	doJobTask = iota
)

// Queue data (queries, responses, etc)
type queue struct {
	id       string
	jobs     chan *queueJob
	task     chan int
	ask      chan struct{}
	response chan struct{}
	quit     chan struct{}
}

// queueJob produces a task which contains query/response and status (done)
type queueJob struct {
	done   chan struct{}
	query  chan []byte
	method string
	answer chan *http.Response
}

// queueBundle is the bundle for the queue data (queries, responses, etc)
type queueBundle struct {
	mutex   sync.Mutex
	records map[string]*queue
}

// checks the queue, if it does not exist, creates it
func (bundle *queueBundle) check(id string) (*queue, bool) {
	bundle.mutex.Lock()
	defer bundle.mutex.Unlock()

	// check for a new record
	_, ok := bundle.records[id]

	// if it is new
	if !ok {
		bundle.records[id] = &queue{
			id:       id,
			jobs:     make(chan *queueJob, MaxJobs),
			task:     make(chan int, MaxJobs),
			ask:      make(chan struct{}, cmdQueueCapacity),
			response: make(chan struct{}, cmdQueueCapacity),
			quit:     make(chan struct{}),
		}
		return bundle.records[id], false
	}

	// if it already exists
	return bundle.records[id], true
}

// removes the queue and stops the worker
func (bundle *queueBundle) remove(id string, timeout time.Duration) {
	bundle.mutex.Lock()
	defer bundle.mutex.Unlock()

	// if a queue exists, the worker must be stopped and a queue must be deleted
	if q, ok := bundle.records[id]; ok {

		// if the worker is alive
		if getResponse(q, timeout) {

			// send a 'quit' command to the worker
			q.quit <- struct{}{}

			// get a response from the worker
			<-q.response
		}

		// delete the queue
		delete(bundle.records, id)
	}
}

// getReponse method is waiting a response or get the false value if timeout
func getResponse(q *queue, timeout time.Duration) bool {
	ticker := time.NewTimer(time.Second * timeout)

	// sweeps unwanted responses if exist
	for {
		select {
		case <-q.response:
			continue
		default:
		}
		break
	}

	// Sends an ASK to the worker
	q.ask <- struct{}{}

	select {
	// If no response is received, should exit via timeout (worker is not alive)
	case <-ticker.C:
		// sweeps ask which sent before (if exist)
		select {
		case <-q.ask:
		default:
		}
		return false
	// After the response has been received, exit
	case <-q.response:
		return true
	}
}
