package spawn

import (
	"testing"
)

func TestQueue(t *testing.T) {
	bundle := &queueBundle{records: make(map[string]*queue)}

	// create new queue
	q, ok := bundle.check("test")
	test(t, !ok, "Expected create new queue, got than exists")

	// start simplest worker
	go func(q *queue) {
		defer func() { q.response <- struct{}{} }()
		for {
			select {
			case <-q.quit:
				return
			case <-q.ask:
				q.response <- struct{}{}
			}
		}
	}(q)

	// get correct response
	status := getResponse(q, 1)
	test(t, status, "Expected get response without timeout, got with")

	// remove the queue
	bundle.remove("test", 1)

	// get correct response
	q, ok = bundle.records["test"]
	test(t, !ok, "Expected queue must be deleted, got the queue exists")
}
