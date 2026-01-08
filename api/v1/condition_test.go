package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("OperationStatus Deletion Utility Functions", func() {

	Describe("ShouldMarkJobAsDeleted", func() {
		Context("when operation has no conditions", func() {
			It("should return false for shouldMarkDeleted and isDone", func() {
				status := &OperationStatus{Conditions: []OperationCondition{}}
				shouldMark, isDone := status.ShouldMarkJobAsDeleted()
				Expect(shouldMark).To(BeFalse())
				Expect(isDone).To(BeFalse())
			})
		})

		Context("when operation is in Starting state", func() {
			It("should allow job creation", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationStarting, corev1.ConditionTrue, "test", "test"),
					},
				}
				shouldMark, isDone := status.ShouldMarkJobAsDeleted()
				Expect(shouldMark).To(BeFalse())
				Expect(isDone).To(BeFalse())
			})
		})

		Context("when operation is in Running state", func() {
			It("should mark as deleted", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationRunning, corev1.ConditionTrue, "test", "test"),
					},
				}
				shouldMark, isDone := status.ShouldMarkJobAsDeleted()
				Expect(shouldMark).To(BeTrue())
				Expect(isDone).To(BeFalse())
			})
		})

		Context("when operation is in Warning state", func() {
			It("should mark as deleted", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationWarning, corev1.ConditionTrue, "PodUnschedulable", "Pod cannot be scheduled"),
					},
				}
				shouldMark, isDone := status.ShouldMarkJobAsDeleted()
				Expect(shouldMark).To(BeTrue())
				Expect(isDone).To(BeFalse())
			})
		})

		Context("when operation is in Succeeded state", func() {
			It("should indicate operation is done", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationSucceeded, corev1.ConditionTrue, "test", "test"),
					},
				}
				shouldMark, isDone := status.ShouldMarkJobAsDeleted()
				Expect(shouldMark).To(BeFalse())
				Expect(isDone).To(BeTrue())
			})
		})

		Context("when operation is in Failed state", func() {
			It("should indicate operation is done", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationFailed, corev1.ConditionTrue, "JobFailed", "Job failed"),
					},
				}
				shouldMark, isDone := status.ShouldMarkJobAsDeleted()
				Expect(shouldMark).To(BeFalse())
				Expect(isDone).To(BeTrue())
			})
		})

		Context("when operation is in Stopped state", func() {
			It("should indicate operation is done", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationStopped, corev1.ConditionTrue, "UserStopped", "Stopped by user"),
					},
				}
				shouldMark, isDone := status.ShouldMarkJobAsDeleted()
				Expect(shouldMark).To(BeFalse())
				Expect(isDone).To(BeTrue())
			})
		})

		Context("when operation has multiple conditions", func() {
			It("should only check the last condition - Running", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationStarting, corev1.ConditionTrue, "test", "test"),
						NewOperationCondition(OperationRunning, corev1.ConditionTrue, "test", "test"),
					},
				}
				shouldMark, isDone := status.ShouldMarkJobAsDeleted()
				Expect(shouldMark).To(BeTrue())
				Expect(isDone).To(BeFalse())
			})

			It("should only check the last condition - Succeeded after Running", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationStarting, corev1.ConditionTrue, "test", "test"),
						NewOperationCondition(OperationRunning, corev1.ConditionTrue, "test", "test"),
						NewOperationCondition(OperationSucceeded, corev1.ConditionTrue, "test", "test"),
					},
				}
				shouldMark, isDone := status.ShouldMarkJobAsDeleted()
				Expect(shouldMark).To(BeFalse())
				Expect(isDone).To(BeTrue())
			})
		})

		Context("when operation has many conditions", func() {
			It("should handle large condition arrays efficiently", func() {
				conditions := make([]OperationCondition, 100)
				for i := 0; i < 99; i++ {
					conditions[i] = NewOperationCondition(OperationStarting, corev1.ConditionTrue, "test", "test")
				}
				conditions[99] = NewOperationCondition(OperationRunning, corev1.ConditionTrue, "test", "test")

				status := &OperationStatus{Conditions: conditions}
				shouldMark, isDone := status.ShouldMarkJobAsDeleted()
				Expect(shouldMark).To(BeTrue())
				Expect(isDone).To(BeFalse())
			})
		})

		Context("when operation transitions from Starting to Warning", func() {
			It("should mark as deleted if last state is Warning", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationStarting, corev1.ConditionTrue, "test", "test"),
						NewOperationCondition(OperationWarning, corev1.ConditionTrue, "ImagePullError", "Cannot pull image"),
					},
				}
				shouldMark, isDone := status.ShouldMarkJobAsDeleted()
				Expect(shouldMark).To(BeTrue())
				Expect(isDone).To(BeFalse())
			})
		})
	})

	Describe("MarkJobAsDeleted", func() {
		Context("when marking as failed", func() {
			It("should set Failed status with correct message", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationRunning, corev1.ConditionTrue, "test", "test"),
					},
				}
				updated := status.MarkJobAsDeleted("Kubernetes Job", true)

				Expect(updated).To(BeTrue())
				Expect(status.IsFailed()).To(BeTrue())
				Expect(status.CompletionTime).NotTo(BeNil())
				Expect(status.Conditions[len(status.Conditions)-1].Type).To(Equal(OperationFailed))
				Expect(status.Conditions[len(status.Conditions)-1].Reason).To(Equal("JobDeleted"))
				Expect(status.Conditions[len(status.Conditions)-1].Message).To(Equal("The underlying Kubernetes Job was deleted externally"))
			})

			It("should set completion time", func() {
				beforeTime := metav1.Now()
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationRunning, corev1.ConditionTrue, "test", "test"),
					},
				}
				status.MarkJobAsDeleted("Kubernetes Job", true)
				afterTime := metav1.Now()

				Expect(status.CompletionTime).NotTo(BeNil())
				Expect(status.CompletionTime.Time).To(BeTemporally(">=", beforeTime.Time))
				Expect(status.CompletionTime.Time).To(BeTemporally("<=", afterTime.Time))
			})
		})

		Context("when marking as stopped", func() {
			It("should set Stopped status with correct message", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationRunning, corev1.ConditionTrue, "test", "test"),
					},
				}
				updated := status.MarkJobAsDeleted("TFJob", false)

				Expect(updated).To(BeTrue())
				Expect(status.IsStopped()).To(BeTrue())
				Expect(status.CompletionTime).NotTo(BeNil())
				Expect(status.Conditions[len(status.Conditions)-1].Type).To(Equal(OperationStopped))
				Expect(status.Conditions[len(status.Conditions)-1].Reason).To(Equal("JobDeleted"))
				Expect(status.Conditions[len(status.Conditions)-1].Message).To(Equal("The underlying TFJob was deleted externally"))
			})
		})

		Context("when marking different job types", func() {
			It("should include correct job type in message - PyTorchJob", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationRunning, corev1.ConditionTrue, "test", "test"),
					},
				}
				status.MarkJobAsDeleted("PyTorchJob", false)

				Expect(status.Conditions[len(status.Conditions)-1].Message).To(Equal("The underlying PyTorchJob was deleted externally"))
			})

			It("should include correct job type in message - MPIJob", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationRunning, corev1.ConditionTrue, "test", "test"),
					},
				}
				status.MarkJobAsDeleted("MPIJob", false)

				Expect(status.Conditions[len(status.Conditions)-1].Message).To(Equal("The underlying MPIJob was deleted externally"))
			})
		})

		Context("when status is already in terminal state", func() {
			It("should update to Failed even if already Succeeded", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationSucceeded, corev1.ConditionTrue, "test", "test"),
					},
				}
				updated := status.MarkJobAsDeleted("Kubernetes Job", true)

				// Since we're logging a new condition, it should update
				Expect(updated).To(BeTrue())
				Expect(status.IsFailed()).To(BeTrue())
			})

			It("should not update if already marked with same reason", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationFailed, corev1.ConditionTrue, "JobDeleted", "The underlying Kubernetes Job was deleted externally"),
					},
				}
				completionTime := metav1.Now()
				status.CompletionTime = &completionTime

				updated := status.MarkJobAsDeleted("Kubernetes Job", true)

				// Should not update if condition is identical
				Expect(updated).To(BeFalse())
			})
		})

		Context("when starting from Warning state", func() {
			It("should successfully transition to Failed", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationWarning, corev1.ConditionTrue, "PodUnschedulable", "Pod cannot be scheduled"),
					},
				}
				updated := status.MarkJobAsDeleted("Kubernetes Job", true)

				Expect(updated).To(BeTrue())
				Expect(status.IsFailed()).To(BeTrue())
			})

			It("should successfully transition to Stopped", func() {
				status := &OperationStatus{
					Conditions: []OperationCondition{
						NewOperationCondition(OperationWarning, corev1.ConditionTrue, "PodUnschedulable", "Pod cannot be scheduled"),
					},
				}
				updated := status.MarkJobAsDeleted("TFJob", false)

				Expect(updated).To(BeTrue())
				Expect(status.IsStopped()).To(BeTrue())
			})
		})

		Context("with empty status", func() {
			It("should handle empty conditions array", func() {
				status := &OperationStatus{Conditions: []OperationCondition{}}
				updated := status.MarkJobAsDeleted("Kubernetes Job", true)

				Expect(updated).To(BeTrue())
				Expect(status.IsFailed()).To(BeTrue())
				Expect(status.Conditions).To(HaveLen(1))
			})
		})
	})
})
