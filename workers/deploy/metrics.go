package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	// deploysTotal counts total deploys processed, labeled by status.
	deploysTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nidus_deploys_total",
			Help: "Total number of deploys processed",
		},
		[]string{"status"},
	)

	// deployDuration tracks full deploy duration (git + build + container).
	deployDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nidus_deploy_duration_seconds",
			Help:    "Deploy duration in seconds",
			Buckets: []float64{5, 10, 20, 30, 60, 120, 300, 600},
		},
	)

	// deployActive tracks currently running deploys.
	deployActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nidus_deploy_active",
			Help: "Number of deploys currently being processed",
		},
	)

	// buildDuration tracks Docker build duration only.
	buildDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nidus_build_duration_seconds",
			Help:    "Docker build duration in seconds",
			Buckets: []float64{5, 10, 20, 30, 60, 120, 300},
		},
	)

	// gitDuration tracks git operations duration.
	gitDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nidus_git_duration_seconds",
			Help:    "Git operations duration in seconds",
			Buckets: []float64{1, 2, 5, 10, 20, 30},
		},
	)

	// buildQueueSize tracks the current build queue size.
	buildQueueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nidus_build_queue_size",
			Help: "Current size of the build queue",
		},
	)

	// buildCacheHits counts Docker build cache hits.
	buildCacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nidus_build_cache_hits_total",
			Help: "Total number of Docker build cache hits",
		},
	)

	// buildCacheMisses counts Docker build cache misses.
	buildCacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nidus_build_cache_misses_total",
			Help: "Total number of Docker build cache misses",
		},
	)
)

func init() {
	prometheus.MustRegister(
		deploysTotal, deployDuration, deployActive,
		buildDuration, gitDuration, buildQueueSize,
		buildCacheHits, buildCacheMisses,
	)
}

// MetricsHandler returns an http.Handler that serves Prometheus metrics.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
