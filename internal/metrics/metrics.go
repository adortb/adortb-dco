package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RenderDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dco_render_duration_seconds",
		Help:    "DCO render pipeline latency",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25},
	}, []string{"status"})

	RenderTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dco_render_total",
		Help: "Total DCO render requests",
	}, []string{"status"})
)

func Handler() http.Handler {
	return promhttp.Handler()
}
