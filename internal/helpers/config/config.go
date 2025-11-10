package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	// Namespace touse
	Namespace = "POLYAXON_K8S_NAMESPACE"

	// SingleNamespace is a flag to enable watching all namespaces
	SingleNamespace = "POLYAXON_SINGLE_NAMESPACE"

	// Max concurrent reconciles configuration
	MaxConcurrentReconciles = "POLYAXON_MAX_CONCURRENT_RECONCILES"

	// Leader election configuration
	LeaderElection = "POLYAXON_LEADER_ELECTION"

	// TFJobEnabled is a flag to enable TFJob conroller
	TFJobEnabled = "POLYAXON_TFJOB_ENABLED"

	// PytorchJobEnabled is a flag to enable PytorchJob conroller
	PytorchJobEnabled = "POLYAXON_PYTORCH_JOB_ENABLED"

	// MPIJobEnabled is a flag to enable MPIJob conroller
	MPIJobEnabled = "POLYAXON_MPIJOB_ENABLED"

	// DaskClusterEnabled is a flag to enable Dask Cluster conroller
	DaskClusterEnabled = "POLYAXON_DASK_CLUSTER_ENABLED"

	// RayClusterEnabled is a flag to enable Ray conroller
	RayClusterEnabled = "POLYAXON_RAY_JOB_ENABLED"

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

	// If agent is enabled
	AgentEnabled = "POLYAXON_SET_AGENT"

	// Log level
	LogLevel = "POLYAXON_LOG_LEVEL"

	// EnableStatusFinalizers to use finalizer
	EnableStatusFinalizers = "POLYAXON_ENABLE_STATUS_FINALIZERS"

	// EnableLogsFinalizers to use logs finalizer
	EnableLogsFinalizers = "POLYAXON_ENABLE_LOGS_FINALIZERS"

	PolyaxonRunIdLabel = "operation.polyaxon.com/uuid"
)

// GetStrEnv returns an environment str variable given by key or return a default value.
func GetStrEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

// GetBoolEnv returns an environment bool variable given by key or return a default value.
// Accepts: "true", "True", "TRUE", "1" as true values
func GetBoolEnv(key string, defaultValue bool) bool {
	value := strings.ToLower(GetStrEnv(key, "false"))
	if value == "true" || value == "1" {
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
