package ops

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	defaultLabels           = []string{"request_type"}
	successPluginServerCall = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "multicluster_clusterversion_success_plugin_call_total",
			Help: "Number of success call for plugin server",
		},
		defaultLabels,
	)
	failedPluginServerCall = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "multicluster_clusterversion_failed_plugin_call_total",
			Help: "Number of failed call for plugin server",
		},
		defaultLabels,
	)
)

func addSuccessPluginServerCall(request string) {
	successPluginServerCall.With(prometheus.Labels{"request_type": request}).Inc()
}

func addFailedPluginServerCall(request string) {
	failedPluginServerCall.With(prometheus.Labels{"request_type": request}).Inc()
}

func init() {
	metrics.Registry.MustRegister(successPluginServerCall, failedPluginServerCall)
}
