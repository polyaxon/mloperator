package config

import (
	"os"
	"strconv"
)

const (
	// Namespace is a flag to enable TFJob conroller
	Namespace = "POLYAXON_K8S_NAMESPACE"

	// Max concurrent reconciles configuration
	MaxConcurrentReconciles = "POLYAXON_MAX_CONCURRENT_RECONCILES"

	// Leader election configuration
	LeaderElection = "POLYAXON_LEADER_ELECTION"

	// TFJobEnabled is a flag to enable TFJob conroller
	TFJobEnabled = "POLYAXON_TFJOB_ENABLED"

	// PytorchJobEnabled is a flag to enable PytorchJob conroller
	PytorchJobEnabled = "POLYAXON_PYTORCH_JOB_ENABLED"

	// PaddleJobEnabled is a flag to enable PaddleJob conroller
	PaddleJobEnabled = "POLYAXON_PADDLE_JOB_ENABLED"

	// MPIJobEnabled is a flag to enable MPIJob conroller
	MPIJobEnabled = "POLYAXON_MPIJOB_ENABLED"

	// MXJobEnabled is a flag to enable MPIJob conroller
	MXJobEnabled = "POLYAXON_MXJOB_ENABLED"

	// XGBoostJobEnabled is a flag to enable MPIJob conroller
	XGBoostJobEnabled = "POLYAXON_XGBOOST_JOB_ENABLED"

	// SparkJobEnabled is a flag to enable Spark conroller
	SparkJobEnabled = "POLYAXON_SPARK_JOB_ENABLED"

	// IstioEnabled is a flag to enable istio controller
	IstioEnabled = "POLYAXON_ISTIO_ENABLED"

	// IstioGateway is the istio gateway to use
	IstioGateway = "POLYAXON_ISTIO_GATEWAY"

	// IstioTLSMode is the istio tls mode to use
	IstioTLSMode = "POLYAXON_ISTIO_TLS_MODE"

	// IstioPrefix is the istio tls mode to use
	IstioPrefix = "POLYAXON_ISTIO_PREFIX"

	// IstioTimeout is the istio default timeout
	IstioTimeout = "POLYAXON_ISTIO_TIMEOUT"

	// ClusterDomain is the istio tls mode to use
	ClusterDomain = "POLYAXON_CLUSTER_DOMAIN"

	// ProxyServicesPort port serving services
	ProxyServicesPort = "POLYAXON_PROXY_SERVICES_PORT"

	// If agent is enabled
	AgentEnabled = "POLYAXON_SET_AGENT"

	// Log level
	LogLevel = "POLYAXON_LOG_LEVEL"
)

// GetStrEnv returns an environment str variable given by key or return a default value.
func GetStrEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

// GetBoolEnv returns an environment bool variable given by key or return a default value.
func GetBoolEnv(key string, defaultValue bool) bool {
	if GetStrEnv(key, "false") == "true" {
		return true
	}
	return defaultValue
}

// GetIntEnv returns an environment int variable given by key or return a default value.
func GetIntEnv(key string, defaultValue int) int {
	if valueStr, ok := os.LookupEnv(key); ok {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}
