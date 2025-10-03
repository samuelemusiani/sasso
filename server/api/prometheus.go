package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sasso_http_requests_total",
			Help: "Total number of HTTP requests.",
		}, []string{"method", "code", "path"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sasso_http_request_duration_seconds",
			Help:    "Histogram of latencies for HTTP requests.",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
		}, []string{"method", "code", "path"},
	)
)

func prometheusHandler(path string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			now := time.Now()

			defer func() {
				ls := []string{r.Method, strconv.Itoa(ww.Status()), path}
				httpRequestsTotal.WithLabelValues(ls...).Inc()
				requestDuration.WithLabelValues(ls...).Observe(time.Since(now).Seconds())
			}()

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}
