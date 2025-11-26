package service

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiv1 "github.com/polyaxon/mloperator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
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
})
