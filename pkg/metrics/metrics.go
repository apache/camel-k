package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"strings"
)

var (
	integration = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "camel_k_integration_phase",
			Help: "Number of integration processed",
		}, []string{
			"phase",
			"id",
		},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(integration)
}

func UpdateIntegrationPhase(iId string, p string) {
	phase := strings.Replace(strings.ToLower(p), " ", "_", -1)

	if phase != "" && iId != "" {
		labels := prometheus.Labels{
			"id":    iId,
			"phase": phase,
		}
		integration.With(labels).Inc()
	}
}
