package job

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
)

var _ = Describe("Job Controller - Deleted Job Detection", func() {
	var (
		reconciler    *JobReconciler
		ctx           context.Context
		fakeClient    client.Client
		scheme        *runtime.Scheme
		instance      *apiv1.Job
		testNamespace string
	)

	BeforeEach(func() {
		ctx = context.Background()
		testNamespace = "default"

		// Setup scheme
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(batchv1.AddToScheme(scheme)).To(Succeed())
		Expect(apiv1.AddToScheme(scheme)).To(Succeed())

		// Setup fake client with status subresource support
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&apiv1.Job{}).
			Build()

		// Setup reconciler
		reconciler = &JobReconciler{
			Client: fakeClient,
			Log:    zap.New(zap.UseDevMode(true)),
			Scheme: scheme,
		}
	})

	Context("when job doesn't exist and was never created", func() {
		It("should create a new job", func() {
			instance = &apiv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-new",
					Namespace: testNamespace,
				},
				BatchJobSpec: &apiv1.BatchJobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
				Termination: apiv1.TerminationSpec{},
				Status: apiv1.OperationStatus{
					Conditions: []apiv1.OperationCondition{},
				},
			}

			// Create the instance in the fake client
			Expect(fakeClient.Create(ctx, instance)).To(Succeed())

			// Reconcile
			err := reconciler.reconcileJob(ctx, instance)
			Expect(err).NotTo(HaveOccurred())

			// Verify job was created
			foundJob := &batchv1.Job{}
			err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-job-new", Namespace: testNamespace}, foundJob)
			Expect(err).NotTo(HaveOccurred())
			Expect(foundJob.Name).To(Equal("test-job-new"))
		})
	})

	Context("when job was running but is now deleted", func() {
		It("should mark operation as stopped", func() {
			instance = &apiv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-deleted",
					Namespace: testNamespace,
				},
				BatchJobSpec: &apiv1.BatchJobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
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

			// Create the instance (but NOT the underlying job - simulating deletion)
			Expect(fakeClient.Create(ctx, instance)).To(Succeed())

			// Reconcile
			err := reconciler.reconcileJob(ctx, instance)
			Expect(err).NotTo(HaveOccurred())

			// Verify status was marked as stopped
			updatedInstance := &apiv1.Job{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{Name: "test-job-deleted", Namespace: testNamespace}, updatedInstance)).To(Succeed())
			Expect(updatedInstance.Status.IsStopped()).To(BeTrue())
			Expect(updatedInstance.Status.CompletionTime).NotTo(BeNil())

			// Verify the reason and message
			lastCondition := updatedInstance.Status.Conditions[len(updatedInstance.Status.Conditions)-1]
			Expect(lastCondition.Reason).To(Equal("JobDeleted"))
			Expect(lastCondition.Message).To(ContainSubstring("Kubernetes Job was deleted externally"))
		})

		It("should not recreate the job", func() {
			instance = &apiv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-no-recreate",
					Namespace: testNamespace,
				},
				BatchJobSpec: &apiv1.BatchJobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
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
			err := reconciler.reconcileJob(ctx, instance)
			Expect(err).NotTo(HaveOccurred())

			// Verify job was NOT created
			foundJob := &batchv1.Job{}
			err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-job-no-recreate", Namespace: testNamespace}, foundJob)
			Expect(err).To(HaveOccurred())
			Expect(apierrs.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("when job had warnings but is now deleted", func() {
		It("should mark operation as stopped", func() {
			instance = &apiv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-warning-deleted",
					Namespace: testNamespace,
				},
				BatchJobSpec: &apiv1.BatchJobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
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
			err := reconciler.reconcileJob(ctx, instance)
			Expect(err).NotTo(HaveOccurred())

			// Verify status was marked as stopped
			updatedInstance := &apiv1.Job{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{Name: "test-job-warning-deleted", Namespace: testNamespace}, updatedInstance)).To(Succeed())
			Expect(updatedInstance.Status.IsStopped()).To(BeTrue())
		})
	})

	Context("when job is in Starting state but not created yet", func() {
		It("should create the job", func() {
			instance = &apiv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-starting",
					Namespace: testNamespace,
				},
				BatchJobSpec: &apiv1.BatchJobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
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
			err := reconciler.reconcileJob(ctx, instance)
			Expect(err).NotTo(HaveOccurred())

			// Verify job was created (not marked as deleted)
			foundJob := &batchv1.Job{}
			err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-job-starting", Namespace: testNamespace}, foundJob)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when job is already in terminal state", func() {
		It("should not recreate job when Succeeded", func() {
			instance = &apiv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-succeeded",
					Namespace: testNamespace,
				},
				BatchJobSpec: &apiv1.BatchJobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
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

			initialConditionsLen := len(instance.Status.Conditions)

			// Reconcile
			err := reconciler.reconcileJob(ctx, instance)
			Expect(err).NotTo(HaveOccurred())

			// Verify no status change and job not created
			updatedInstance := &apiv1.Job{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{Name: "test-job-succeeded", Namespace: testNamespace}, updatedInstance)).To(Succeed())
			Expect(len(updatedInstance.Status.Conditions)).To(Equal(initialConditionsLen))
			Expect(updatedInstance.Status.IsSucceeded()).To(BeTrue())

			// Verify job was not created
			foundJob := &batchv1.Job{}
			err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-job-succeeded", Namespace: testNamespace}, foundJob)
			Expect(err).To(HaveOccurred())
			Expect(apierrs.IsNotFound(err)).To(BeTrue())
		})

		It("should not recreate job when Failed", func() {
			instance = &apiv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-failed",
					Namespace: testNamespace,
				},
				BatchJobSpec: &apiv1.BatchJobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
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
			err := reconciler.reconcileJob(ctx, instance)
			Expect(err).NotTo(HaveOccurred())

			// Verify job was not created
			foundJob := &batchv1.Job{}
			err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-job-failed", Namespace: testNamespace}, foundJob)
			Expect(err).To(HaveOccurred())
			Expect(apierrs.IsNotFound(err)).To(BeTrue())
		})

		It("should not recreate job when Stopped", func() {
			instance = &apiv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-stopped",
					Namespace: testNamespace,
				},
				BatchJobSpec: &apiv1.BatchJobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
				Termination: apiv1.TerminationSpec{},
				Status: apiv1.OperationStatus{
					Conditions: []apiv1.OperationCondition{
						apiv1.NewOperationCondition(apiv1.OperationStopped, corev1.ConditionTrue, "UserStopped", "Stopped by user"),
					},
				},
			}

			Expect(fakeClient.Create(ctx, instance)).To(Succeed())

			// Reconcile
			err := reconciler.reconcileJob(ctx, instance)
			Expect(err).NotTo(HaveOccurred())

			// Verify job was not created
			foundJob := &batchv1.Job{}
			err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-job-stopped", Namespace: testNamespace}, foundJob)
			Expect(err).To(HaveOccurred())
			Expect(apierrs.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("when job exists and is running", func() {
		It("should update job status", func() {
			instance = &apiv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-exists",
					Namespace: testNamespace,
				},
				BatchJobSpec: &apiv1.BatchJobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
				Termination: apiv1.TerminationSpec{},
				Status: apiv1.OperationStatus{
					Conditions: []apiv1.OperationCondition{},
				},
			}

			// Create the instance
			Expect(fakeClient.Create(ctx, instance)).To(Succeed())

			// First reconcile - should create the job
			err := reconciler.reconcileJob(ctx, instance)
			Expect(err).NotTo(HaveOccurred())

			// Verify job was created
			foundJob := &batchv1.Job{}
			err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-job-exists", Namespace: testNamespace}, foundJob)
			Expect(err).NotTo(HaveOccurred())

			// Set owner reference for the created job
			Expect(ctrl.SetControllerReference(instance, foundJob, scheme)).To(Succeed())
			Expect(fakeClient.Update(ctx, foundJob)).To(Succeed())

			// Second reconcile - job exists
			err = reconciler.reconcileJob(ctx, instance)
			Expect(err).NotTo(HaveOccurred())

			// Verify job still exists
			err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-job-exists", Namespace: testNamespace}, foundJob)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when job has multiple condition transitions", func() {
		It("should handle Starting -> Running -> Deleted sequence", func() {
			instance = &apiv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-transitions",
					Namespace: testNamespace,
				},
				BatchJobSpec: &apiv1.BatchJobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
				Termination: apiv1.TerminationSpec{},
				Status: apiv1.OperationStatus{
					Conditions: []apiv1.OperationCondition{
						apiv1.NewOperationCondition(apiv1.OperationStarting, corev1.ConditionTrue, "test", "test"),
						apiv1.NewOperationCondition(apiv1.OperationRunning, corev1.ConditionTrue, "test", "test"),
					},
				},
			}

			Expect(fakeClient.Create(ctx, instance)).To(Succeed())

			// Reconcile with job deleted
			err := reconciler.reconcileJob(ctx, instance)
			Expect(err).NotTo(HaveOccurred())

			// Verify status is now stopped (last condition was Running)
			updatedInstance := &apiv1.Job{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{Name: "test-job-transitions", Namespace: testNamespace}, updatedInstance)).To(Succeed())
			Expect(updatedInstance.Status.IsStopped()).To(BeTrue())
		})
	})
})
