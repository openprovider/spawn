// Copyright 2015 Igor Dolzhikov. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package spawn

import (
	"net/http"
	"sync"
	"time"
)

const (
	cmdQueueCapacity = 1

	checkHealthTask = iota
	doJobTask
	repeatJobTask
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

// queueJob produce task with query/response and status (done)
type queueJob struct {
	done   bool
	query  chan []byte
	answer chan *http.Response
}

// queueBundle is a bundle for the queue data (queries, responses, etc)
type queueBundle struct {
	mutex   sync.Mutex
	records map[string]*queue
}

// check a queue, if it does not exist, create it
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
			quit:     make(chan struct{}, cmdQueueCapacity),
		}
		return bundle.records[id], false
	}

	// if it exists already
	return bundle.records[id], true
}

// remove the queue and stops the worker
func (bundle *queueBundle) remove(id string, timeout time.Duration) {
	bundle.mutex.Lock()
	defer bundle.mutex.Unlock()

	// if a queue exists, the worker must be stoped and a queue must be deleted
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

// getReponse method is waiting a response or get the false value by timeout
func getResponse(q *queue, timeout time.Duration) bool {
	ticker := time.NewTimer(time.Second * timeout)

	// a unwanted ask/response sweeps if exist
	for {
		select {
		case <-q.response:
			continue
		case <-q.ask:
			continue
		default:
		}
		break
	}

	// Sends an ASK to the worker
	q.ask <- struct{}{}

	select {
	// Exit by timeout if a response does not get (worker is not alive)
	case <-ticker.C:
		// a unwanted ask sweeps if exists
		for {
			select {
			case <-q.ask:
				continue
			default:
			}
			break
		}
		return false
	// Exit after a response received
	case <-q.response:
		return true
	}
}
