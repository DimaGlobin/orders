package transport

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler exposes the Prometheus default registry over HTTP.
// All metric collectors are registered via promauto in the metrics package,
// so this handler is stateless.
type MetricsHandler struct{}

func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

func (h *MetricsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("GET /metrics", promhttp.Handler())
}
