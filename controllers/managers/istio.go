package managers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/polyaxon/mloperator/controllers/config"
	"github.com/polyaxon/mloperator/controllers/kinds"
)

const (
	// DefaultTimeout for service
	DefaultTimeout = "300s"
)

// CopyVirtualService copies the owned fields from one VirtualService to another
// Returns true if the fields copied from don't match to.
func CopyVirtualService(from, to *unstructured.Unstructured) bool {
	return CopyUnstructuredSpec(from, to, []string{"spec"})
}

// GenerateVirtualService returns a batch job given a OperationSpec
func GenerateVirtualService(name, namespace string) (*unstructured.Unstructured, error) {
	istioPrefis := config.GetStrEnv(config.IstioPrefix, "/")
	prefix := fmt.Sprintf("/%s/%s/%s/", istioPrefis, namespace, name)
	rewrite := fmt.Sprintf("/%s/%s/%s/", istioPrefis, namespace, name)
	clusterDomain := config.GetStrEnv(config.ClusterDomain, "cluster.local")
	service := fmt.Sprintf("%s.%s.svc.%s", name, namespace, clusterDomain)

	virtualService := &unstructured.Unstructured{}
	virtualService.SetAPIVersion(kinds.IstioAPIVersion)
	virtualService.SetKind(kinds.IstioVirtualServiceKind)
	virtualService.SetName(name)
	virtualService.SetNamespace(namespace)
	if err := unstructured.SetNestedStringSlice(virtualService.Object, []string{"*"}, "spec", "hosts"); err != nil {
		return nil, fmt.Errorf("Set .spec.hosts error: %v", err)
	}

	istioGateway := config.GetStrEnv(config.IstioGateway, "polyaxon/mloperator-gateway")

	if err := unstructured.SetNestedStringSlice(virtualService.Object, []string{istioGateway},
		"spec", "gateways"); err != nil {
		return nil, fmt.Errorf("Set .spec.gateways error: %v", err)
	}

	http := []interface{}{
		map[string]interface{}{
			"match": []interface{}{
				map[string]interface{}{
					"uri": map[string]interface{}{
						"prefix": prefix,
					},
				},
			},
			"rewrite": map[string]interface{}{
				"uri": rewrite,
			},
			"route": []interface{}{
				map[string]interface{}{
					"destination": map[string]interface{}{
						"host": service,
						"port": map[string]interface{}{
							"number": int64(DefaultServingPort),
						},
					},
				},
			},
			"timeout": config.GetStrEnv(config.IstioTimeout, DefaultTimeout),
		},
	}
	if err := unstructured.SetNestedSlice(virtualService.Object, http, "spec", "http"); err != nil {
		return nil, fmt.Errorf("Set .spec.http error: %v", err)
	}

	return virtualService, nil

}
