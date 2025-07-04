package provider

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type ProviderStoreMetrics struct {
	TasksComplete      prometheus.HistogramVec // request duration with status and offloading target
	ConnectedProviders prometheus.GaugeFunc    // currently connected providers
	ConnectedWorkers   prometheus.GaugeFunc    // currently connected workers (sum over all providers)
}

func (store *ProviderStore) initializePrometheusMetrics() {
	if store.metrics != nil {
		return // initialized already
	}
	m := &ProviderStoreMetrics{}
	store.metrics = m

	// track request durations with their success status and target
	m.TasksComplete = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "wasimoff_completed_tasks",
		Help:    "Completed tasks with their status and offloading target.",
		Buckets: prometheus.DefBuckets,
	}, []string{"status", "target"})

	// currently connected providers
	m.ConnectedProviders = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "wasimoff_conn_providers",
		Help: "Currently connected Providers.",
	}, func() float64 {
		return float64(store.Size())
	})

	// currently connected workers across all providers
	m.ConnectedWorkers = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "wasimoff_conn_providers_workers",
		Help: "Sum of Workers across currently connected Providers.",
	}, func() float64 {
		sum := 0
		store.Range(func(addr string, provider *Provider) bool {
			sum += provider.CurrentLimit()
			return true
		})
		return float64(sum)
	})

}

// Observe a completed task and update histogram metric
func (s *ProviderStore) Observe(task *AsyncTask) {

	target := targetProvider
	if task.CloudOffloaded {
		target = targetCloud
	}

	status := statusErr
	if task.Error == nil {
		s.ratecounter.Observe()
		status = statusOk
	}

	s.metrics.TasksComplete.With(prometheus.Labels{
		"status": status,
		"target": target,
	}).Observe(time.Since(task.start).Seconds())

}

const (
	statusOk  = "ok"
	statusErr = "err"
)

const (
	targetProvider = "provider"
	targetCloud    = "cloud"
)
