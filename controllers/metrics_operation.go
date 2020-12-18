package controllers

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	defaultLabels    = []string{"operation"}
	successOperation = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "multicluster_clusterversion_success_operation_total",
			Help: "Number of performed success cluster operations",
		},
		defaultLabels,
	)
	failedOperation = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "multicluster_clusterversion_failed_operation_total",
			Help: "Number of performed failed cluster operations",
		},
		defaultLabels,
	)
)

func addSuccessOperation(operation string) {
	successOperation.With(prometheus.Labels{"operation": operation}).Inc()
}

func addFailedOperation(operation string) {
	failedOperation.With(prometheus.Labels{"operation": operation}).Inc()
}

func init() {
	metrics.Registry.MustRegister(
		successOperation,
		failedOperation,
	)
}
