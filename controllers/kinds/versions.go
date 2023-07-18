package kinds

const (
	// KFAPIVersion api version
	KFAPIVersion = "kubeflow.org/v1"

	// MPIJobKind kind
	MPIJobKind = "MPIJob"

	// TFJobKind kind
	TFJobKind = "TFJob"

	// PytorchJobKind kind
	PytorchJobKind = "PyTorchJob"

	// MXJobKind kind
	MXJobKind = "MXJob"

	// PaddleJobKind kind
	PaddleJobKind = "PaddleJob"

	// XGBoostJobKind tfjob kind
	XGBoostJobKind = "XGBoostJob"

	// IstioAPIVersion istio networing api version
	IstioAPIVersion = "networking.istio.io/v1alpha3"

	// IstioVirtualServiceKind istio virtual service kind
	IstioVirtualServiceKind = "VirtualService"

	// SparkAPIVersion Spark operator api version
	SparkAPIVersion = "sparkoperator.k8s.io/v1beta2"

	// SparkApplicationKind Spark application kind
	SparkApplicationKind = "SparkApplication"

	// DaskAPIVersion Dask operator api version
	DaskAPIVersion = "kubernetes.dask.org/v1"

	// DaskJobKind Dask job kind
	DaskJobKind = "DaskJob"

	// RayAPIVersion Ray operator api version
	RayAPIVersion = "ray.io/v1alpha1"

	// RayJobKind Ray job kind
	RayJobKind = "RayJob"
)
