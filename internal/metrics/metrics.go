// Package metrics exposes Prometheus instrumentation for Raptor.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// RequestsCaptured counts captured inbound requests, labelled by type.
	RequestsCaptured = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "raptor_requests_captured_total",
		Help: "Total inbound requests captured, by type (web|email|dns).",
	}, []string{"type"})

	// RequestsRejected counts requests rejected before capture, by reason.
	RequestsRejected = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "raptor_requests_rejected_total",
		Help: "Total inbound requests rejected before capture, by reason.",
	}, []string{"reason"})
)

func init() {
	prometheus.MustRegister(RequestsCaptured, RequestsRejected)
}

// Handler returns the Prometheus metrics HTTP handler for /metrics.
func Handler() http.Handler { return promhttp.Handler() }
