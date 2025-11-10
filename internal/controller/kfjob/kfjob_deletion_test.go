package kfjob

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
	"github.com/polyaxon/mloperator/internal/controller/kfjob/kinds"
)

var _ = Describe("KfJob Controller - Deleted Job Detection", func() {
	var (
		reconciler    *KfJobReconciler
		ctx           context.Context
		fakeClient    client.Client
		scheme        *runtime.Scheme
		instance      *apiv1.KfJob
		testNamespace string
	)

	BeforeEach(func() {
		ctx = context.Background()
		testNamespace = "default"

		// Setup scheme
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(apiv1.AddToScheme(scheme)).To(Succeed())

		// Setup fake client with status subresource support
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&apiv1.KfJob{}).
			Build()

		// Setup reconciler
		reconciler = &KfJobReconciler{
			Client: fakeClient,
			Log:    zap.New(zap.UseDevMode(true)),
			Scheme: scheme,
		}
	})

	Describe("TFJob", func() {
		Context("when TFJob was running but is now deleted", func() {
			It("should mark operation as stopped", func() {
				instance = &apiv1.KfJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-tfjob-deleted",
						Namespace: testNamespace,
					},
					TFJobSpec: &apiv1.TFJobSpec{
						ReplicaSpecs: map[apiv1.TFReplicaType]*apiv1.KFReplicaSpec{
							apiv1.TFReplicaTypeWorker: {
								Replicas: int32Ptr(1),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:  "tensorflow",
												Image: "tensorflow:latest",
											},
										},
									},
								},
							},
						},
					},
					Termination: apiv1.TerminationSpec{},
					Status: apiv1.OperationStatus{
						Conditions: []apiv1.OperationCondition{
							apiv1.NewOperationCondition(apiv1.OperationRunning, corev1.ConditionTrue, "test", "test"),
						},
					},
				}

				// Create the instance (but NOT the underlying TFJob - simulating deletion)
				Expect(fakeClient.Create(ctx, instance)).To(Succeed())

				// Reconcile
				err := reconciler.reconcileTFJob(ctx, instance)
				Expect(err).NotTo(HaveOccurred())

				// Verify status was marked as stopped
				updatedInstance := &apiv1.KfJob{}
				Expect(fakeClient.Get(ctx, types.NamespacedName{Name: "test-tfjob-deleted", Namespace: testNamespace}, updatedInstance)).To(Succeed())
				Expect(updatedInstance.Status.IsStopped()).To(BeTrue())
				Expect(updatedInstance.Status.CompletionTime).NotTo(BeNil())

				// Verify the reason and message
				lastCondition := updatedInstance.Status.Conditions[len(updatedInstance.Status.Conditions)-1]
				Expect(lastCondition.Reason).To(Equal("JobDeleted"))
				Expect(lastCondition.Message).To(ContainSubstring("TFJob was deleted externally"))
			})

			It("should not recreate the TFJob", func() {
				instance = &apiv1.KfJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-tfjob-no-recreate",
						Namespace: testNamespace,
					},
					TFJobSpec: &apiv1.TFJobSpec{
						ReplicaSpecs: map[apiv1.TFReplicaType]*apiv1.KFReplicaSpec{
							apiv1.TFReplicaTypeWorker: {
								Replicas: int32Ptr(1),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:  "tensorflow",
												Image: "tensorflow:latest",
											},
										},
									},
								},
							},
						},
					},
					Termination: apiv1.TerminationSpec{},
					Status: apiv1.OperationStatus{
						Conditions: []apiv1.OperationCondition{
							apiv1.NewOperationCondition(apiv1.OperationRunning, corev1.ConditionTrue, "test", "test"),
						},
					},
				}

				Expect(fakeClient.Create(ctx, instance)).To(Succeed())

				// Reconcile
				err := reconciler.reconcileTFJob(ctx, instance)
				Expect(err).NotTo(HaveOccurred())

				// Verify TFJob was NOT created
				foundJob := &unstructured.Unstructured{}
				foundJob.SetAPIVersion(kinds.KFAPIVersion)
				foundJob.SetKind(kinds.TFJobKind)
				err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-tfjob-no-recreate", Namespace: testNamespace}, foundJob)
				Expect(err).To(HaveOccurred())
				Expect(apierrs.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("when TFJob had warnings but is now deleted", func() {
			It("should mark operation as stopped", func() {
				instance = &apiv1.KfJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-tfjob-warning-deleted",
						Namespace: testNamespace,
					},
					TFJobSpec: &apiv1.TFJobSpec{
						ReplicaSpecs: map[apiv1.TFReplicaType]*apiv1.KFReplicaSpec{
							apiv1.TFReplicaTypeWorker: {
								Replicas: int32Ptr(1),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:  "tensorflow",
												Image: "tensorflow:latest",
											},
										},
									},
								},
							},
						},
					},
					Termination: apiv1.TerminationSpec{},
					Status: apiv1.OperationStatus{
						Conditions: []apiv1.OperationCondition{
							apiv1.NewOperationCondition(apiv1.OperationWarning, corev1.ConditionTrue, "PodUnschedulable", "Pod cannot be scheduled"),
						},
					},
				}

				Expect(fakeClient.Create(ctx, instance)).To(Succeed())

				// Reconcile
				err := reconciler.reconcileTFJob(ctx, instance)
				Expect(err).NotTo(HaveOccurred())

				// Verify status was marked as stopped
				updatedInstance := &apiv1.KfJob{}
				Expect(fakeClient.Get(ctx, types.NamespacedName{Name: "test-tfjob-warning-deleted", Namespace: testNamespace}, updatedInstance)).To(Succeed())
				Expect(updatedInstance.Status.IsStopped()).To(BeTrue())
			})
		})

		Context("when TFJob is in Starting state but not created yet", func() {
			It("should create the TFJob", func() {
				instance = &apiv1.KfJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-tfjob-starting",
						Namespace: testNamespace,
					},
					TFJobSpec: &apiv1.TFJobSpec{
						ReplicaSpecs: map[apiv1.TFReplicaType]*apiv1.KFReplicaSpec{
							apiv1.TFReplicaTypeWorker: {
								Replicas: int32Ptr(1),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:  "tensorflow",
												Image: "tensorflow:latest",
											},
										},
									},
								},
							},
						},
					},
					Termination: apiv1.TerminationSpec{},
					Status: apiv1.OperationStatus{
						Conditions: []apiv1.OperationCondition{
							apiv1.NewOperationCondition(apiv1.OperationStarting, corev1.ConditionTrue, "test", "test"),
						},
					},
				}

				Expect(fakeClient.Create(ctx, instance)).To(Succeed())

				// Reconcile
				err := reconciler.reconcileTFJob(ctx, instance)
				Expect(err).NotTo(HaveOccurred())

				// Verify TFJob was created (not marked as deleted)
				foundJob := &unstructured.Unstructured{}
				foundJob.SetAPIVersion(kinds.KFAPIVersion)
				foundJob.SetKind(kinds.TFJobKind)
				err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-tfjob-starting", Namespace: testNamespace}, foundJob)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when TFJob is already in terminal state", func() {
			It("should not recreate TFJob when Succeeded", func() {
				instance = &apiv1.KfJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-tfjob-succeeded",
						Namespace: testNamespace,
					},
					TFJobSpec: &apiv1.TFJobSpec{
						ReplicaSpecs: map[apiv1.TFReplicaType]*apiv1.KFReplicaSpec{
							apiv1.TFReplicaTypeWorker: {
								Replicas: int32Ptr(1),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:  "tensorflow",
												Image: "tensorflow:latest",
											},
										},
									},
								},
							},
						},
					},
					Termination: apiv1.TerminationSpec{},
					Status: apiv1.OperationStatus{
						Conditions: []apiv1.OperationCondition{
							apiv1.NewOperationCondition(apiv1.OperationSucceeded, corev1.ConditionTrue, "test", "test"),
						},
					},
				}

				Expect(fakeClient.Create(ctx, instance)).To(Succeed())

				// Reconcile
				err := reconciler.reconcileTFJob(ctx, instance)
				Expect(err).NotTo(HaveOccurred())

				// Verify TFJob was not created
				foundJob := &unstructured.Unstructured{}
				foundJob.SetAPIVersion(kinds.KFAPIVersion)
				foundJob.SetKind(kinds.TFJobKind)
				err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-tfjob-succeeded", Namespace: testNamespace}, foundJob)
				Expect(err).To(HaveOccurred())
				Expect(apierrs.IsNotFound(err)).To(BeTrue())
			})
		})
	})

	Describe("PyTorchJob", func() {
		Context("when PyTorchJob was running but is now deleted", func() {
			It("should mark operation as stopped", func() {
				instance = &apiv1.KfJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pytorchjob-deleted",
						Namespace: testNamespace,
					},
					PytorchJobSpec: &apiv1.PytorchJobSpec{
						ReplicaSpecs: map[apiv1.PyTorchReplicaType]*apiv1.KFReplicaSpec{
							apiv1.PyTorchReplicaTypeMaster: {
								Replicas: int32Ptr(1),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:  "pytorch",
												Image: "pytorch:latest",
											},
										},
									},
								},
							},
						},
					},
					Termination: apiv1.TerminationSpec{},
					Status: apiv1.OperationStatus{
						Conditions: []apiv1.OperationCondition{
							apiv1.NewOperationCondition(apiv1.OperationRunning, corev1.ConditionTrue, "test", "test"),
						},
					},
				}

				Expect(fakeClient.Create(ctx, instance)).To(Succeed())

				// Reconcile
				err := reconciler.reconcilePytorchJob(ctx, instance)
				Expect(err).NotTo(HaveOccurred())

				// Verify status was marked as stopped
				updatedInstance := &apiv1.KfJob{}
				Expect(fakeClient.Get(ctx, types.NamespacedName{Name: "test-pytorchjob-deleted", Namespace: testNamespace}, updatedInstance)).To(Succeed())
				Expect(updatedInstance.Status.IsStopped()).To(BeTrue())

				// Verify the message contains correct job type
				lastCondition := updatedInstance.Status.Conditions[len(updatedInstance.Status.Conditions)-1]
				Expect(lastCondition.Message).To(ContainSubstring("PyTorchJob was deleted externally"))
			})
		})

		Context("when PyTorchJob had warnings but is now deleted", func() {
			It("should mark operation as stopped", func() {
				instance = &apiv1.KfJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pytorchjob-warning",
						Namespace: testNamespace,
					},
					PytorchJobSpec: &apiv1.PytorchJobSpec{
						ReplicaSpecs: map[apiv1.PyTorchReplicaType]*apiv1.KFReplicaSpec{
							apiv1.PyTorchReplicaTypeMaster: {
								Replicas: int32Ptr(1),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:  "pytorch",
												Image: "pytorch:latest",
											},
										},
									},
								},
							},
						},
					},
					Termination: apiv1.TerminationSpec{},
					Status: apiv1.OperationStatus{
						Conditions: []apiv1.OperationCondition{
							apiv1.NewOperationCondition(apiv1.OperationWarning, corev1.ConditionTrue, "ImagePullError", "Cannot pull image"),
						},
					},
				}

				Expect(fakeClient.Create(ctx, instance)).To(Succeed())

				// Reconcile
				err := reconciler.reconcilePytorchJob(ctx, instance)
				Expect(err).NotTo(HaveOccurred())

				// Verify status was marked as stopped
				updatedInstance := &apiv1.KfJob{}
				Expect(fakeClient.Get(ctx, types.NamespacedName{Name: "test-pytorchjob-warning", Namespace: testNamespace}, updatedInstance)).To(Succeed())
				Expect(updatedInstance.Status.IsStopped()).To(BeTrue())
			})
		})
	})

	Describe("MPIJob", func() {
		Context("when MPIJob was running but is now deleted", func() {
			It("should mark operation as stopped", func() {
				instance = &apiv1.KfJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-mpijob-deleted",
						Namespace: testNamespace,
					},
					MPIJobSpec: &apiv1.MPIJobSpec{
						ReplicaSpecs: map[apiv1.MPIReplicaType]*apiv1.KFReplicaSpec{
							apiv1.MPIReplicaTypeLauncher: {
								Replicas: int32Ptr(1),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:  "mpi",
												Image: "mpi:latest",
											},
										},
									},
								},
							},
						},
					},
					Termination: apiv1.TerminationSpec{},
					Status: apiv1.OperationStatus{
						Conditions: []apiv1.OperationCondition{
							apiv1.NewOperationCondition(apiv1.OperationRunning, corev1.ConditionTrue, "test", "test"),
						},
					},
				}

				Expect(fakeClient.Create(ctx, instance)).To(Succeed())

				// Reconcile
				err := reconciler.reconcileMPIJob(ctx, instance)
				Expect(err).NotTo(HaveOccurred())

				// Verify status was marked as stopped
				updatedInstance := &apiv1.KfJob{}
				Expect(fakeClient.Get(ctx, types.NamespacedName{Name: "test-mpijob-deleted", Namespace: testNamespace}, updatedInstance)).To(Succeed())
				Expect(updatedInstance.Status.IsStopped()).To(BeTrue())

				// Verify the message contains correct job type
				lastCondition := updatedInstance.Status.Conditions[len(updatedInstance.Status.Conditions)-1]
				Expect(lastCondition.Message).To(ContainSubstring("MPIJob was deleted externally"))
			})
		})

		Context("when MPIJob is already Failed", func() {
			It("should not recreate MPIJob", func() {
				instance = &apiv1.KfJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-mpijob-failed",
						Namespace: testNamespace,
					},
					MPIJobSpec: &apiv1.MPIJobSpec{
						ReplicaSpecs: map[apiv1.MPIReplicaType]*apiv1.KFReplicaSpec{
							apiv1.MPIReplicaTypeLauncher: {
								Replicas: int32Ptr(1),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:  "mpi",
												Image: "mpi:latest",
											},
										},
									},
								},
							},
						},
					},
					Termination: apiv1.TerminationSpec{},
					Status: apiv1.OperationStatus{
						Conditions: []apiv1.OperationCondition{
							apiv1.NewOperationCondition(apiv1.OperationFailed, corev1.ConditionTrue, "JobFailed", "Job failed"),
						},
					},
				}

				Expect(fakeClient.Create(ctx, instance)).To(Succeed())

				// Reconcile
				err := reconciler.reconcileMPIJob(ctx, instance)
				Expect(err).NotTo(HaveOccurred())

				// Verify MPIJob was not created
				foundJob := &unstructured.Unstructured{}
				foundJob.SetAPIVersion(kinds.KFAPIVersion)
				foundJob.SetKind(kinds.MPIJobKind)
				err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-mpijob-failed", Namespace: testNamespace}, foundJob)
				Expect(err).To(HaveOccurred())
				Expect(apierrs.IsNotFound(err)).To(BeTrue())
			})
		})
	})
})

// Helper function to create int32 pointer
func int32Ptr(i int32) *int32 {
	return &i
}
