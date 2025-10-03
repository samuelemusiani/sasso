package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		}, []string{"method", "code"},
	)
)

func prometheusHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(promhttp.InstrumentHandlerCounter(httpRequestsTotal, h))
}
