package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helpers", func() {

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Helpers", func() {
		It("should contain string", func() {
			slice1 := []string{"foo", "bar", "moo"}
			slice2 := []string{"foo2", "boo2", "moo2"}

			for _, str := range slice1 {
				Expect(containsString(slice1, str)).To(BeTrue())
			}

			for _, str := range slice2 {
				Expect(containsString(slice1, str)).To(BeFalse())
			}
		})

		It("should remove string", func() {
			slice := []string{"foo", "bar", "moo"}
			before := len(slice)

			for _, str := range slice {
				Expect(containsString(removeString(slice, str), str)).To(BeFalse())
			}

			for _, str := range slice {
				Expect(len(slice)).To(BeIdenticalTo(before))
				slice = removeString(slice, str)
				Expect(len(slice)).To(BeIdenticalTo(before - 1))
				before--
			}

			Expect(len(slice)).To(BeIdenticalTo(0))
		})
	})
})
