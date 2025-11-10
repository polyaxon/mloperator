package main

import (
	"context"
	"crypto/tls"
	"flag"
	"os"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// mpijobv1 "github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v1"
	// pytorchjobv1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1"
	// tfjobv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/controller/cluster"
	"github.com/polyaxon/mloperator/internal/controller/job"
	"github.com/polyaxon/mloperator/internal/controller/kfjob"
	"github.com/polyaxon/mloperator/internal/controller/service"

	"github.com/polyaxon/mloperator/internal/helpers/config"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
	utilruntime.Must(apiv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// registerUnstructuredIndex registers a field index for unstructured resources
func registerUnstructuredIndex(mgr ctrl.Manager, gvk schema.GroupVersionKind, indexField string, ownerKind string) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)

	return mgr.GetFieldIndexer().IndexField(context.Background(), u, indexField, func(rawObj client.Object) []string {
		obj := rawObj.(*unstructured.Unstructured)
		owner := metav1.GetControllerOf(obj)
		if owner == nil {
			return nil
		}
		// Check if the owner is a Polyaxon resource of the expected kind
		if owner.APIVersion != apiv1.GroupVersion.String() || owner.Kind != ownerKind {
			return nil
		}
		return []string{owner.Name}
	})
}

// nolint:gocyclo
func main() {
	var metricsAddr string
	var metricsCertPath, metricsCertName, metricsCertKey string
	var webhookCertPath, webhookCertName, webhookCertKey string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var tlsOpts []func(*tls.Config)
	var cacheOpts cache.Options
	var namespace string
	var debugMode bool

	// Allow to pass by env and override by flag
	if config.GetBoolEnv(config.SingleNamespace, false) {
		if config.GetStrEnv(config.Namespace, "") != "" {
			namespace = config.GetStrEnv(config.Namespace, "")
		} else {
			namespace = "polyaxon"
		}
	} else {
		namespace = ""
	}
	if config.GetBoolEnv(config.LeaderElection, false) {
		enableLeaderElection = true
	} else {
		enableLeaderElection = false
	}
	if strings.ToLower(config.GetStrEnv(config.LogLevel, "")) == "debug" {
		debugMode = true
	} else {
		debugMode = false
	}
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&namespace, "namespace", namespace, "The namespace to restrict the operator.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.StringVar(&webhookCertPath, "webhook-cert-path", "", "The directory that contains the webhook certificate.")
	flag.StringVar(&webhookCertName, "webhook-cert-name", "tls.crt", "The name of the webhook certificate file.")
	flag.StringVar(&webhookCertKey, "webhook-cert-key", "tls.key", "The name of the webhook key file.")
	flag.StringVar(&metricsCertPath, "metrics-cert-path", "",
		"The directory that contains the metrics server certificate.")
	flag.StringVar(&metricsCertName, "metrics-cert-name", "tls.crt", "The name of the metrics server certificate file.")
	flag.StringVar(&metricsCertKey, "metrics-cert-key", "tls.key", "The name of the metrics server key file.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	opts := zap.Options{
		Development: debugMode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	// Initial webhook TLS options
	webhookTLSOpts := tlsOpts
	webhookServerOptions := webhook.Options{
		TLSOpts: webhookTLSOpts,
	}

	if len(webhookCertPath) > 0 {
		setupLog.Info("Initializing webhook certificate watcher using provided certificates",
			"webhook-cert-path", webhookCertPath, "webhook-cert-name", webhookCertName, "webhook-cert-key", webhookCertKey)

		webhookServerOptions.CertDir = webhookCertPath
		webhookServerOptions.CertName = webhookCertName
		webhookServerOptions.KeyName = webhookCertKey
	}

	webhookServer := webhook.NewServer(webhookServerOptions)

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	// If the certificate is not specified, controller-runtime will automatically
	// generate self-signed certificates for the metrics server. While convenient for development and testing,
	// this setup is not recommended for production.
	//
	// TODO(user): If you enable certManager, uncomment the following lines:
	// - [METRICS-WITH-CERTS] at config/default/kustomization.yaml to generate and use certificates
	// managed by cert-manager for the metrics server.
	// - [PROMETHEUS-WITH-CERTS] at config/prometheus/kustomization.yaml for TLS certification.
	if len(metricsCertPath) > 0 {
		setupLog.Info("Initializing metrics certificate watcher using provided certificates",
			"metrics-cert-path", metricsCertPath, "metrics-cert-name", metricsCertName, "metrics-cert-key", metricsCertKey)

		metricsServerOptions.CertDir = metricsCertPath
		metricsServerOptions.CertName = metricsCertName
		metricsServerOptions.KeyName = metricsCertKey
	}

	if namespace != "" {
		cacheOpts = cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Cache:                  cacheOpts,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "operations-polyaxon-controller",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Register field indexes once for all controllers
	indexOwnerField := ".metadata.controller"
	indexEventInvolvedObjectUidField := ".involvedObject.uid"
	podUIDIndexField := "metadata.uid"

	// Index Event by `involvedObject.uid`
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Event{}, indexEventInvolvedObjectUidField, func(rawObj client.Object) []string {
		event := rawObj.(*corev1.Event)
		if event.InvolvedObject.UID == "" {
			return nil
		}
		return []string{string(event.InvolvedObject.UID)}
	}); err != nil {
		setupLog.Error(err, "unable to create index for Event")
		os.Exit(1)
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, podUIDIndexField, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{string(pod.UID)}
	}); err != nil {
		setupLog.Error(err, "unable to create index for Pod")
		os.Exit(1)
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &batchv1.Job{}, indexOwnerField, func(rawObj client.Object) []string {
		job := rawObj.(*batchv1.Job)
		owner := metav1.GetControllerOf(job)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != apiv1.GroupVersion.String() || owner.Kind != "job" {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		setupLog.Error(err, "unable to create index for Job")
		os.Exit(1)
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.Deployment{}, indexOwnerField, func(rawObj client.Object) []string {
		deployment := rawObj.(*appsv1.Deployment)
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != apiv1.GroupVersion.String() || owner.Kind != "service" {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		setupLog.Error(err, "unable to create index for Deployment")
		os.Exit(1)
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, indexOwnerField, func(rawObj client.Object) []string {
		service := rawObj.(*corev1.Service)
		owner := metav1.GetControllerOf(service)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != apiv1.GroupVersion.String() || owner.Kind != "service" {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		setupLog.Error(err, "unable to create index for Service")
		os.Exit(1)
	}

	// Register indexes for Kubeflow job types (conditionally based on enabled flags)
	kubeflowJobTypes := []schema.GroupVersionKind{}

	if config.GetBoolEnv(config.TFJobEnabled, false) {
		kubeflowJobTypes = append(kubeflowJobTypes, schema.GroupVersionKind{Group: "kubeflow.org", Version: "v1", Kind: "TFJob"})
	}
	if config.GetBoolEnv(config.PytorchJobEnabled, false) {
		kubeflowJobTypes = append(kubeflowJobTypes, schema.GroupVersionKind{Group: "kubeflow.org", Version: "v1", Kind: "PyTorchJob"})
	}
	if config.GetBoolEnv(config.MPIJobEnabled, false) {
		kubeflowJobTypes = append(kubeflowJobTypes, schema.GroupVersionKind{Group: "kubeflow.org", Version: "v1", Kind: "MPIJob"})
	}

	for _, gvk := range kubeflowJobTypes {
		if err := registerUnstructuredIndex(mgr, gvk, indexOwnerField, "kfjob"); err != nil {
			setupLog.Error(err, "unable to create index for unstructured resource", "kind", gvk.Kind)
			os.Exit(1)
		}
	}

	// Register indexes for cluster types (conditionally based on enabled flags)
	clusterTypes := []schema.GroupVersionKind{}

	if config.GetBoolEnv(config.RayClusterEnabled, false) {
		clusterTypes = append(clusterTypes, schema.GroupVersionKind{Group: "ray.io", Version: "v1", Kind: "RayCluster"})
	}
	if config.GetBoolEnv(config.DaskClusterEnabled, false) {
		clusterTypes = append(clusterTypes, schema.GroupVersionKind{Group: "kubernetes.dask.org", Version: "v1", Kind: "DaskCluster"})
	}
	// Note: SparkApplication doesn't have a corresponding config flag yet
	// Uncomment when POLYAXON_SPARK_JOB_ENABLED is added to config
	// if config.GetBoolEnv(config.SparkJobEnabled, false) {
	//     clusterTypes = append(clusterTypes, schema.GroupVersionKind{Group: "sparkoperator.k8s.io", Version: "v1beta2", Kind: "SparkApplication"})
	// }

	for _, gvk := range clusterTypes {
		if err := registerUnstructuredIndex(mgr, gvk, indexOwnerField, "cluster"); err != nil {
			setupLog.Error(err, "unable to create index for unstructured resource", "kind", gvk.Kind)
			os.Exit(1)
		}
	}

	if err := (&job.JobReconciler{
		Client:    mgr.GetClient(),
		Log:       ctrl.Log.WithName("controllers").WithName("Job"),
		Scheme:    mgr.GetScheme(),
		Namespace: namespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Job")
		os.Exit(1)
	}
	if err := (&service.ServiceReconciler{
		Client:    mgr.GetClient(),
		Log:       ctrl.Log.WithName("controllers").WithName("Service"),
		Scheme:    mgr.GetScheme(),
		Namespace: namespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Service")
		os.Exit(1)
	}
	if err := (&kfjob.KfJobReconciler{
		Client:    mgr.GetClient(),
		Log:       ctrl.Log.WithName("controllers").WithName("KfJob"),
		Scheme:    mgr.GetScheme(),
		Namespace: namespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KfJob")
		os.Exit(1)
	}
	if err := (&cluster.ClusterReconciler{
		Client:    mgr.GetClient(),
		Log:       ctrl.Log.WithName("controllers").WithName("Cluster"),
		Scheme:    mgr.GetScheme(),
		Namespace: namespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cluster")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
