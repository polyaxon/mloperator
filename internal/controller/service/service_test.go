package service

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiv1 "github.com/polyaxon/mloperator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Service Controller", func() {
	var (
		ctx        context.Context
		reconciler *ServiceReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()
		reconciler = &ServiceReconciler{
			Client: k8sClient,
			Log:    ctrl.Log.WithName("controllers").WithName("Service"),
			Scheme: k8sClient.Scheme(),
		}
	})

	Context("When handling culling", func() {
		It("should not cull if culling is not enabled", func() {
			instance := &apiv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service-no-culling",
					Namespace: "default",
				},
				Termination: apiv1.TerminationSpec{},
				Status: apiv1.OperationStatus{
					Conditions: []apiv1.OperationCondition{
						{
							Type: apiv1.OperationRunning,
						},
					},
				},
			}

			duration, err := reconciler.handleCulling(ctx, instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(duration).To(BeNil())
		})

		It("should not cull if service is not running", func() {
			timeout := int32(3600)
			instance := &apiv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service-not-running",
					Namespace: "default",
				},
				Termination: apiv1.TerminationSpec{
					Culling: &apiv1.CullingSpec{
						Timeout: &timeout,
					},
				},
				Status: apiv1.OperationStatus{
					Conditions: []apiv1.OperationCondition{
						{
							Type: apiv1.OperationStarting,
						},
					},
				},
			}

			duration, err := reconciler.handleCulling(ctx, instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(duration).To(BeNil())
		})

		It("should requeue if service is running and probe fails", func() {
			timeout := int32(3600)
			instance := &apiv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service-running-probe-fail",
					Namespace: "default",
				},
				Termination: apiv1.TerminationSpec{
					Culling: &apiv1.CullingSpec{
						Timeout: &timeout,
					},
					Probe: &apiv1.ActivityProbe{
						Http: &apiv1.ActivityProbeHttp{
							Port: 8888,
							Path: "/",
						},
					},
				},
				Status: apiv1.OperationStatus{
					Conditions: []apiv1.OperationCondition{
						{
							Type:   apiv1.OperationRunning,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}

			duration, err := reconciler.handleCulling(ctx, instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(duration).NotTo(BeNil())
			Expect(*duration).To(Equal(time.Duration(timeout) * time.Second))
		})

		It("should use default service port if probe port is not set", func() {
			timeout := int32(3600)
			instance := &apiv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service-default-port",
					Namespace: "default",
				},
				Termination: apiv1.TerminationSpec{
					Culling: &apiv1.CullingSpec{
						Timeout: &timeout,
					},
					Probe: &apiv1.ActivityProbe{
						Http: &apiv1.ActivityProbeHttp{
							// Port not set
							Path: "/",
						},
					},
				},
				ServiceSpec: &apiv1.ServiceSpec{
					Ports: []int32{8888},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "test-container",
								},
							},
						},
					},
				},
				Status: apiv1.OperationStatus{
					Conditions: []apiv1.OperationCondition{
						{
							Type:   apiv1.OperationRunning,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}

			duration, err := reconciler.handleCulling(ctx, instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(duration).NotTo(BeNil())
			Expect(*duration).To(Equal(time.Duration(timeout) * time.Second))
		})

		It("should handle custom path with trailing slash", func() {
			timeout := int32(3600)
			instance := &apiv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service-custom-path",
					Namespace: "default",
				},
				Termination: apiv1.TerminationSpec{
					Culling: &apiv1.CullingSpec{
						Timeout: &timeout,
					},
					Probe: &apiv1.ActivityProbe{
						Http: &apiv1.ActivityProbeHttp{
							Port: 8888,
							Path: "/lab/",
						},
					},
				},
				ServiceSpec: &apiv1.ServiceSpec{
					Ports: []int32{8888},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "test-container",
								},
							},
						},
					},
				},
				Status: apiv1.OperationStatus{
					Conditions: []apiv1.OperationCondition{
						{
							Type:   apiv1.OperationRunning,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}

			duration, err := reconciler.handleCulling(ctx, instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(duration).NotTo(BeNil())
			Expect(*duration).To(Equal(time.Duration(timeout) * time.Second))
		})
	})

	Context("When reconciling deployment status", func() {
		newInstance := func(name string, labels map[string]string) *apiv1.Service {
			return &apiv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: "default",
					Labels:    labels,
				},
				Status: apiv1.OperationStatus{
					Conditions: []apiv1.OperationCondition{
						{
							Type:   apiv1.OperationRunning,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}
		}

		runningDeployment := func() appsv1.Deployment {
			return appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 1,
					ReadyReplicas:     1,
					Conditions: []appsv1.DeploymentCondition{
						{
							Type:    appsv1.DeploymentAvailable,
							Status:  corev1.ConditionTrue,
							Reason:  "MinimumReplicasAvailable",
							Message: "Deployment has minimum availability.",
						},
					},
				},
			}
		}

		failedPod := func(name string, instanceName string) *corev1.Pod {
			return &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/instance": instanceName,
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "main",
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 1,
									Reason:   "Error",
									Message:  "container exited",
								},
							},
						},
					},
				},
			}
		}

		setFakeClient := func(objects ...client.Object) {
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(k8sClient.Scheme()).
				WithObjects(objects...).
				WithStatusSubresource(&apiv1.Service{}).
				Build()
		}

		It("should reconcile deployment state even when the instance label is missing", func() {
			instance := newInstance("test-service-missing-instance-label", nil)
			setFakeClient(instance)

			updated := reconciler.reconcileDeploymentStatus(instance, runningDeployment())

			Expect(updated).To(BeTrue())
			Expect(instance.Status.IsRunning()).To(BeTrue())
		})

		It("should not log failed pods from the deployment status path", func() {
			instance := newInstance(
				"test-service-failed-pod-deployment-status",
				map[string]string{"app.kubernetes.io/instance": "test-service-failed-pod-deployment-status"},
			)
			pod := failedPod("test-service-failed-pod-deployment-status-pod", instance.Name)
			setFakeClient(instance, pod)

			updated := reconciler.reconcileDeploymentStatus(instance, appsv1.Deployment{})

			Expect(updated).To(BeFalse())
			Expect(instance.Status.IsRunning()).To(BeTrue())
			Expect(instance.Status.IsWarning()).To(BeFalse())
		})
	})

	Context("When handling service container exits", func() {
		newInstance := func(name string, backoffLimit *int32) *apiv1.Service {
			return &apiv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/instance": name,
					},
				},
				Termination: apiv1.TerminationSpec{
					BackoffLimit: backoffLimit,
				},
				ServiceSpec: &apiv1.ServiceSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "main",
								},
							},
						},
					},
				},
				Status: apiv1.OperationStatus{
					Conditions: []apiv1.OperationCondition{
						{
							Type:   apiv1.OperationRunning,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}
		}

		newTermination := func(exitCode int32) *corev1.ContainerStateTerminated {
			reason := "Error"
			if exitCode == 0 {
				reason = "Completed"
			}
			return &corev1.ContainerStateTerminated{
				ExitCode: exitCode,
				Reason:   reason,
				Message:  "container exited",
			}
		}

		newPodWithState := func(
			name string,
			instanceName string,
			state corev1.ContainerState,
			lastTermination *corev1.ContainerStateTerminated,
			restartCount int32,
		) *corev1.Pod {
			return &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/instance": instanceName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "main",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "main",
							RestartCount: restartCount,
							State:        state,
							LastTerminationState: corev1.ContainerState{
								Terminated: lastTermination,
							},
						},
					},
				},
			}
		}

		newPod := func(name string, instanceName string, exitCode int32, restartCount int32) *corev1.Pod {
			return newPodWithState(
				name,
				instanceName,
				corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason: "CrashLoopBackOff",
					},
				},
				newTermination(exitCode),
				restartCount,
			)
		}

		setFakeClient := func(instance *apiv1.Service, pods ...*corev1.Pod) {
			objects := make([]client.Object, 0, 1+len(pods))
			objects = append(objects, instance)
			for _, pod := range pods {
				objects = append(objects, pod)
			}
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(k8sClient.Scheme()).
				WithObjects(objects...).
				WithStatusSubresource(&apiv1.Service{}).
				Build()
		}

		It("should ignore services without pods", func() {
			instance := newInstance("test-service-no-pods", nil)
			setFakeClient(instance)

			done, err := reconciler.handleServiceContainerExit(ctx, instance)

			Expect(err).NotTo(HaveOccurred())
			Expect(done).To(BeFalse())
			Expect(instance.Status.IsDone()).To(BeFalse())
			Expect(instance.Status.CompletionTime).To(BeNil())
		})

		It("should ignore running services with no termination history", func() {
			instance := newInstance("test-service-running", nil)
			pod := newPodWithState(
				"test-service-running-pod",
				instance.Name,
				corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
				nil,
				0,
			)
			setFakeClient(instance, pod)

			done, err := reconciler.handleServiceContainerExit(ctx, instance)

			Expect(err).NotTo(HaveOccurred())
			Expect(done).To(BeFalse())
			Expect(instance.Status.IsDone()).To(BeFalse())
			Expect(instance.Status.CompletionTime).To(BeNil())
		})

		It("should mark a single-replica waiting service succeeded after the main container exits successfully", func() {
			instance := newInstance("test-service-succeeded", nil)
			pod := newPod("test-service-succeeded-pod", instance.Name, 0, 1)
			setFakeClient(instance, pod)

			done, err := reconciler.handleServiceContainerExit(ctx, instance)

			Expect(err).NotTo(HaveOccurred())
			Expect(done).To(BeTrue())
			Expect(instance.Status.IsSucceeded()).To(BeTrue())
			Expect(instance.Status.CompletionTime).NotTo(BeNil())
		})

		It("should ignore successful termination history on running services", func() {
			instance := newInstance("test-service-running-succeeded", nil)
			pod := newPodWithState(
				"test-service-running-succeeded-pod",
				instance.Name,
				corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
				newTermination(0),
				1,
			)
			setFakeClient(instance, pod)

			done, err := reconciler.handleServiceContainerExit(ctx, instance)

			Expect(err).NotTo(HaveOccurred())
			Expect(done).To(BeFalse())
			Expect(instance.Status.IsSucceeded()).To(BeFalse())
			Expect(instance.Status.CompletionTime).To(BeNil())
		})

		It("should fail after the first failed exit when backoffLimit is not set", func() {
			instance := newInstance("test-service-default-backoff", nil)
			pod := newPod("test-service-default-backoff-pod", instance.Name, 1, 1)
			setFakeClient(instance, pod)

			done, err := reconciler.handleServiceContainerExit(ctx, instance)

			Expect(err).NotTo(HaveOccurred())
			Expect(done).To(BeTrue())
			Expect(instance.Status.IsFailed()).To(BeTrue())
			Expect(instance.Status.CompletionTime).NotTo(BeNil())
			condition := instance.Status.Conditions[len(instance.Status.Conditions)-1]
			Expect(condition.Message).To(ContainSubstring(`Main container "main" in pod "test-service-default-backoff-pod"`))
			Expect(condition.Message).To(ContainSubstring("exited with code 1 after 1 failed attempt(s), exceeding maxRetries 0"))
			Expect(condition.Message).To(ContainSubstring("container exited"))
		})

		It("should keep running failed exits within the retry budget", func() {
			backoffLimit := int32(1)
			instance := newInstance("test-service-within-backoff", &backoffLimit)
			pod := newPod("test-service-within-backoff-pod", instance.Name, 1, 1)
			setFakeClient(instance, pod)

			done, err := reconciler.handleServiceContainerExit(ctx, instance)

			Expect(err).NotTo(HaveOccurred())
			Expect(done).To(BeFalse())
			Expect(instance.Status.IsFailed()).To(BeFalse())
			Expect(instance.Status.CompletionTime).To(BeNil())
		})

		It("should fail when failed exits exceed the retry budget", func() {
			backoffLimit := int32(1)
			instance := newInstance("test-service-exceeded-backoff", &backoffLimit)
			pod := newPod("test-service-exceeded-backoff-pod", instance.Name, 1, 2)
			setFakeClient(instance, pod)

			done, err := reconciler.handleServiceContainerExit(ctx, instance)

			Expect(err).NotTo(HaveOccurred())
			Expect(done).To(BeTrue())
			Expect(instance.Status.IsFailed()).To(BeTrue())
			Expect(instance.Status.CompletionTime).NotTo(BeNil())
		})

		It("should fail running services with failed restarts over the retry budget", func() {
			backoffLimit := int32(1)
			instance := newInstance("test-service-running-exceeded-backoff", &backoffLimit)
			pod := newPodWithState(
				"test-service-running-exceeded-backoff-pod",
				instance.Name,
				corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
				newTermination(1),
				2,
			)
			setFakeClient(instance, pod)

			done, err := reconciler.handleServiceContainerExit(ctx, instance)

			Expect(err).NotTo(HaveOccurred())
			Expect(done).To(BeTrue())
			Expect(instance.Status.IsFailed()).To(BeTrue())
			Expect(instance.Status.CompletionTime).NotTo(BeNil())
		})

		It("should fail multi-replica services when any pod exceeds the retry budget", func() {
			backoffLimit := int32(1)
			replicas := int32(2)
			instance := newInstance("test-service-multi-backoff", &backoffLimit)
			instance.ServiceSpec.Replicas = &replicas
			failedPod := newPod("test-service-multi-backoff-failed-pod", instance.Name, 1, 2)
			succeededPod := newPod("test-service-multi-backoff-succeeded-pod", instance.Name, 0, 1)
			setFakeClient(instance, failedPod, succeededPod)

			done, err := reconciler.handleServiceContainerExit(ctx, instance)

			Expect(err).NotTo(HaveOccurred())
			Expect(done).To(BeTrue())
			Expect(instance.Status.IsFailed()).To(BeTrue())
			Expect(instance.Status.CompletionTime).NotTo(BeNil())
		})

		It("should not mark multi-replica services succeeded when all pods exit successfully", func() {
			replicas := int32(2)
			instance := newInstance("test-service-multi-succeeded", nil)
			instance.ServiceSpec.Replicas = &replicas
			pod1 := newPod("test-service-multi-succeeded-pod-1", instance.Name, 0, 1)
			pod2 := newPod("test-service-multi-succeeded-pod-2", instance.Name, 0, 1)
			setFakeClient(instance, pod1, pod2)

			done, err := reconciler.handleServiceContainerExit(ctx, instance)

			Expect(err).NotTo(HaveOccurred())
			Expect(done).To(BeFalse())
			Expect(instance.Status.IsSucceeded()).To(BeFalse())
			Expect(instance.Status.CompletionTime).To(BeNil())
		})

		It("should not fail services with unlimited retries", func() {
			backoffLimit := int32(-1)
			instance := newInstance("test-service-unlimited-backoff", &backoffLimit)
			pod := newPod("test-service-unlimited-backoff-pod", instance.Name, 1, 20)
			setFakeClient(instance, pod)

			done, err := reconciler.handleServiceContainerExit(ctx, instance)

			Expect(err).NotTo(HaveOccurred())
			Expect(done).To(BeFalse())
			Expect(instance.Status.IsFailed()).To(BeFalse())
			Expect(instance.Status.CompletionTime).To(BeNil())
		})
	})
})
