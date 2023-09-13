package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	DwsReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "dws_reconciles_total",
			Help: "Number of total reconciles in DWS controller",
		},
	)
)

func init() {
	metrics.Registry.MustRegister(DwsReconcilesTotal)
}
