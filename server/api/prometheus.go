package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sasso_http_requests_total",
		Help: "Total number of HTTP requests",
	})
)

func prometheusHandler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer httpRequestCount.Inc()

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
