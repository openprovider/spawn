package spawn

import (
	"net/http"
	"sync"
	"text/template"

	"github.com/takama/router"
)

const (
	successMetric = "success"
	failureMetric = "failure"
	queuedMetric  = "queued"
)

type Metrics struct {
	Success struct {
		Get    uint64 `json:"get"`
		Set    uint64 `json:"set"`
		Delete uint64 `json:"delete"`
	} `json:"success"`
	Failure struct {
		Get    uint64 `json:"get"`
		Set    uint64 `json:"set"`
		Delete uint64 `json:"delete"`
	} `json:"failure"`
	Queued struct {
		Get    uint64 `json:"get"`
		Set    uint64 `json:"set"`
		Delete uint64 `json:"delete"`
	} `json:"queued"`
}

// MetricsBandle contains an embedded server link and Node records
type MetricsBandle struct {
	// contains filtered or unexported fields
	mutex sync.RWMutex
	*Server
	update  chan metricsJob
	records map[string]Metrics
}

type metricsJob struct {
	id, metricType, method string
}

func (bundle *MetricsBandle) SetMetrics(id, metricType, method string) {

	bundle.update <- metricsJob{
		id:         id,
		metricType: metricType,
		method:     method,
	}
}

// updateMetrics makes exclusive update of the metrics
func (bundle *MetricsBandle) updateMetrics() {

	for {
		update := <-bundle.update

		// If the job is done, unlocks the bundle

		bundle.mutex.RLock()
		metric := bundle.records[update.id]
		bundle.mutex.RUnlock()

		switch update.metricType {
		case successMetric:
			switch update.method {
			case methodGET:
				metric.Queued.Get--
				metric.Success.Get++
			case methodPUT, methodPOST:
				metric.Queued.Set--
				metric.Success.Set++
			case methodDELETE:
				metric.Queued.Delete--
				metric.Success.Delete++
			}
		case failureMetric:
			switch update.method {
			case methodGET:
				metric.Queued.Get--
				metric.Failure.Get++
			case methodPUT, methodPOST:
				metric.Queued.Set--
				metric.Failure.Set++
			case methodDELETE:
				metric.Queued.Delete--
				metric.Failure.Delete++
			}
		case queuedMetric:
			switch update.method {
			case methodGET:
				metric.Queued.Get++
			case methodPUT, methodPOST:
				metric.Queued.Set++
			case methodDELETE:
				metric.Queued.Delete++
			}
		}

		// Locks the bundle for the transaction processing
		bundle.mutex.Lock()
		bundle.records[update.id] = metric
		bundle.mutex.Unlock()
	}
}

// getMetrics - gets all the nodes metrics
func (bundle *MetricsBandle) getMetrics(c *router.Control) {

	bundle.mutex.RLock()
	defer bundle.mutex.RUnlock()

	templ, err := template.New("metricsList").Parse(metricsList)
	if err == nil {
		c.Writer.Header().Add("Content-type", router.MIMETEXT)
		c.Writer.WriteHeader(http.StatusOK)
		templ.Execute(c.Writer, bundle.records)
		return
	}
	errlog.Println(err)
	c.Code(http.StatusOK).Body(bundle.records)
}

var metricsList = `
{{ range $k, $v := . }}
{{ $k }} 
+=======================================================================+
| REQUESTS        |       GET       |       SET       |      DELETE     |
+=======================================================================+
| SUCCESS         | {{ printf "% 15d" $v.Success.Get }} | {{ printf "% 15d" $v.Success.Set }} | {{ printf "% 15d" $v.Success.Delete }} |
+-----------------+-----------------+-----------------+-----------------+
| FAILURE         | {{ printf "% 15d" $v.Failure.Get }} | {{ printf "% 15d" $v.Failure.Set }} | {{ printf "% 15d" $v.Failure.Delete }} |
+-----------------+-----------------+-----------------+-----------------+
| QUEUED          | {{ printf "% 15d" $v.Queued.Get }} | {{ printf "% 15d" $v.Queued.Set }} | {{ printf "% 15d" $v.Queued.Delete }} |
+-----------------+-----------------+-----------------+-----------------+
{{end}}
`
