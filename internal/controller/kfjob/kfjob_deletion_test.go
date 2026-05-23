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
	"github.com/polyaxon/mloperator/internal/controller/kfjob/kfapi"
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

	Describe("common status reconciliation", func() {
		newInstance := func(name string) *apiv1.KfJob {
			return &apiv1.KfJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: testNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance": name,
					},
				},
				Status: apiv1.OperationStatus{
					Conditions: []apiv1.OperationCondition{},
				},
			}
		}

		newKfJobWithCondition := func(name string, condType kfapi.JobConditionType, reason string, message string) unstructured.Unstructured {
			job := unstructured.Unstructured{Object: map[string]interface{}{}}
			job.SetAPIVersion(kinds.KFAPIVersion)
			job.SetKind(kinds.TFJobKind)
			job.SetName(name)
			job.SetNamespace(testNamespace)
			job.Object["status"] = map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":    string(condType),
						"status":  string(corev1.ConditionTrue),
						"reason":  reason,
						"message": message,
					},
				},
			}
			return job
		}

		newTerminatedPod := func(name string, instanceName string) *corev1.Pod {
			return &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: testNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance": instanceName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "trainer"},
					},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "trainer",
							RestartCount: 2,
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 2,
									Reason:   "Error",
									Message:  "bad args",
								},
							},
						},
					},
				},
			}
		}

		newTransientWarningPod := func(name string, instanceName string) *corev1.Pod {
			return &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: testNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance": instanceName,
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					Conditions: []corev1.PodCondition{
						{
							Type:    corev1.PodReady,
							Status:  corev1.ConditionFalse,
							Reason:  "ContainersNotReady",
							Message: "containers are not ready",
						},
					},
				},
			}
		}

		setFakeClient := func(objects ...client.Object) {
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				WithStatusSubresource(&apiv1.KfJob{}).
				Build()
		}

		It("should enrich failed status with main container termination details", func() {
			instance := newInstance("test-kfjob-failed-details")
			pod := newTerminatedPod("test-kfjob-failed-details-pod", instance.Name)
			setFakeClient(instance, pod)
			job := newKfJobWithCondition(instance.Name, kfapi.JobFailed, "BackoffLimitExceeded", "Job has reached the specified backoff limit")

			updated, err := reconciler.reconcileKfJobStatus(instance, job)

			Expect(err).NotTo(HaveOccurred())
			Expect(updated).To(BeTrue())
			Expect(instance.Status.IsFailed()).To(BeTrue())
			condition := instance.Status.Conditions[len(instance.Status.Conditions)-1]
			Expect(condition.Reason).To(Equal("BackoffLimitExceeded"))
			Expect(condition.Message).To(ContainSubstring("after 3 failed attempt(s)"))
			Expect(condition.Message).To(ContainSubstring(`Main container "trainer" in pod "test-kfjob-failed-details-pod"`))
			Expect(condition.Message).To(ContainSubstring("exit code 2 (Error): bad args"))
		})

		It("should not let transient pod warnings mask failed status", func() {
			instance := newInstance("test-kfjob-failed-over-warning")
			pod := newTransientWarningPod("test-kfjob-failed-over-warning-pod", instance.Name)
			setFakeClient(instance, pod)
			job := newKfJobWithCondition(instance.Name, kfapi.JobFailed, "BackoffLimitExceeded", "Job has reached the specified backoff limit")

			updated, err := reconciler.reconcileKfJobStatus(instance, job)

			Expect(err).NotTo(HaveOccurred())
			Expect(updated).To(BeTrue())
			condition := instance.Status.Conditions[len(instance.Status.Conditions)-1]
			Expect(condition.Type).To(Equal(apiv1.OperationFailed))
			Expect(condition.Reason).To(Equal("BackoffLimitExceeded"))
			Expect(condition.Message).NotTo(ContainSubstring("containers are not ready"))
		})

		It("should not let transient pod warnings mask running status", func() {
			instance := newInstance("test-kfjob-running-over-warning")
			pod := newTransientWarningPod("test-kfjob-running-over-warning-pod", instance.Name)
			setFakeClient(instance, pod)
			job := newKfJobWithCondition(instance.Name, kfapi.JobRunning, "JobRunning", "Job is running")

			updated, err := reconciler.reconcileKfJobStatus(instance, job)

			Expect(err).NotTo(HaveOccurred())
			Expect(updated).To(BeTrue())
			condition := instance.Status.Conditions[len(instance.Status.Conditions)-1]
			Expect(condition.Type).To(Equal(apiv1.OperationRunning))
		})
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
