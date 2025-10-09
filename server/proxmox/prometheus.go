package proxmox

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	workerFunctionsDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sasso_worker_cycle_duration_seconds",
			Help:    "Histogram of latencies for worker cycles.",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
		}, []string{"function"})

	workerCycleDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "sasso_worker_cycle_total_duration_seconds",
			Help:    "Histogram of total latencies for worker cycles.",
			Buckets: prometheus.ExponentialBuckets(0.032, 2, 14),
		})

	objectCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sasso_object_count",
			Help: "Number of objects in the system.",
		}, []string{"object"})
)

func workerCycleDurationObserve(function string, f func()) {
	now := time.Now()
	defer func() {
		workerFunctionsDuration.WithLabelValues(function).Observe(time.Since(now).Seconds())
	}()
	f()
}

func objectCountSet(object string, count int64) {
	objectCount.WithLabelValues(object).Set(float64(count))
}
