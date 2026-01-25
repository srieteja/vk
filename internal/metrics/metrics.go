package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	Inflight        *prometheus.GaugeVec
	once            sync.Once
)

func Init() {
	once.Do(func() {
		RequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		)
		RequestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		)
		Inflight = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "http_requests_inflight",
				Help: "In-flight HTTP requests",
			},
			[]string{"method", "path"},
		)

		prometheus.MustRegister(RequestsTotal, RequestDuration, Inflight)
	})
}
