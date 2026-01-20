package provider

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type ProviderStoreMetrics struct {

	// various histograms for durations in task handling
	TasksCompleted prometheus.HistogramVec // request duration with status and offloading target
	TasksScheduled prometheus.HistogramVec // duration until a task is scheduled
	TasksExecution prometheus.HistogramVec // actual execution time of the task since scheduling

	// count specific events like retries
	TaskRetries prometheus.CounterVec

	// track available resources
	ConnectedProviders        prometheus.GaugeFunc // currently connected providers
	AvailableWorkers          prometheus.GaugeVec  // available workers, partitioned by providers and cloud
	CurrentlyQueuedTasks      prometheus.Gauge     // currently waiting (queued) tasks
	CurrentlyDispatchingTasks prometheus.Gauge     // concurrently scheduling tasks
}

// list of useful histogram buckets
var histogramBuckets = []float64{0.010, 0.100, 0.250, 1, 5, 10, 60, 120, 300}

// create and register all the metric gauges
func (store *ProviderStore) initializePrometheusMetrics() {
	if store.metrics != nil {
		return // initialized already
	}
	m := &ProviderStoreMetrics{}
	store.metrics = m

	// -- histograms

	// track request durations with their success status and target
	m.TasksCompleted = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "wasimoff_tasks_completed",
		Help:    "completed tasks (start to finish); partitioned by status and offloading target",
		Buckets: histogramBuckets,
	}, []string{"status", "target"}) // TODO: add task type (wasi/pyodide) as partition?

	// track durations until task is (re-)scheduled
	m.TasksScheduled = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "wasimoff_tasks_scheduled",
		Help:    "time until task is scheduled (from start); partitioned by offloading target",
		Buckets: histogramBuckets,
	}, []string{"target"})

	// track durations of actual task computation at the provider
	m.TasksExecution = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "wasimoff_tasks_execution",
		Help:    "actual computation time (scheduled to finish); partitioned by status and offloading target",
		Buckets: histogramBuckets,
	}, []string{"status", "target"})

	// -- task event counters

	// number of retries
	m.TaskRetries = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "wasimoff_task_retries_count",
		Help: "number of retries across all scheduled tasks",
	}, []string{"attempt"})

	// currently waiting (queued) and dispatching (scheduling) tasks
	m.CurrentlyQueuedTasks = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "wasimoff_tasks_queued",
		Help: "currently waiting (queued) tasks",
	})
	m.CurrentlyDispatchingTasks = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "wasimoff_tasks_dispatching",
		Help: "currently dispatching (scheduling) tasks",
	})

	// -- available resources

	// total number of available workers across all providers + cloud
	m.AvailableWorkers = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "wasimoff_available_workers",
		Help: "available workers for computation across all providers",
	}, []string{"type"})

	// currently connected providers, which also updates the available worker count
	m.ConnectedProviders = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "wasimoff_connected_providers",
		Help: "currently connected Providers",
	}, func() float64 {
		// return float64(store.Size())
		workers := 0
		providers := 0
		store.Range(func(addr string, provider *Provider) bool {
			providers += 1
			workers += provider.CurrentLimit()
			return true
		})
		m.AvailableWorkers.WithLabelValues("providers").Set(float64(workers))
		return float64(providers)
	})

}

// Observe a retried task to update counter vector
func (s *ProviderStore) ObserveRetry(attempt int) {
	s.metrics.TaskRetries.With(prometheus.Labels{"attempt": fmt.Sprintf("%d", attempt)}).Inc()
}

// Set the current task queue length gauge
func (s *ProviderStore) ObserveTaskQueue(queuelen int, scheduling int) {
	s.metrics.CurrentlyQueuedTasks.Set(float64(queuelen))        // tasks in the queue
	s.metrics.CurrentlyDispatchingTasks.Set(float64(scheduling)) // tasks trying to dispatch
}

// Observe a scheduled task to update historgram
func (s *ProviderStore) ObserveScheduled(task *AsyncTask) {

	target := targetProvider
	if task.CloudOffloaded {
		target = targetCloud
	}

	durScheduled := time.Since(task.TimeStart).Seconds()
	s.metrics.TasksScheduled.With(prometheus.Labels{"target": target}).Observe(durScheduled)

}

// ObserveCompleted a completed task and update histogram metric
func (s *ProviderStore) ObserveCompleted(task *AsyncTask) {

	target := targetProvider
	if task.CloudOffloaded {
		target = targetCloud
	}

	status := statusErr
	if task.Error == nil {
		s.ratecounter.Observe()
		status = statusOk
	}

	durComplete := time.Since(task.TimeStart).Seconds()
	durExecution := time.Since(task.TimeScheduled).Seconds()

	labels := prometheus.Labels{"status": status, "target": target}
	s.metrics.TasksCompleted.With(labels).Observe(durComplete)
	s.metrics.TasksExecution.With(labels).Observe(durExecution)
	// log.Printf("TASK id=%s status=%s on=%s time=%f (%f)", task.Request.GetInfo().GetId(), status, target, durComplete, durExecution)

}

const (
	statusOk  = "ok"
	statusErr = "err"
)

const (
	targetProvider = "provider"
	targetCloud    = "cloud"
)
