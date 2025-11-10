package config

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Config", func() {
		It("Config texts should get default", func() {
			Expect(GetStrEnv("TEST", "")).To(BeIdenticalTo(""))
			Expect(GetStrEnv("test", "test")).To(BeIdenticalTo("test"))
		})

		It("Config texts should get from envs", func() {
			os.Setenv("TEST", "foo")
			Expect(GetStrEnv("TEST", "")).To(BeIdenticalTo("foo"))
			Expect(GetStrEnv("TEST", "test")).To(BeIdenticalTo("foo"))
		})

		It("Config bool should get default", func() {
			Expect(GetBoolEnv("TEST", false)).To(BeFalse())
			Expect(GetBoolEnv("test", true)).To(BeTrue())
		})

		It("Config bool should get from envs", func() {
			os.Setenv("TEST", "true")
			Expect(GetBoolEnv("TEST", false)).To(BeTrue())
			Expect(GetBoolEnv("TEST", true)).To(BeTrue())

			os.Setenv("TEST", "false")
			Expect(GetBoolEnv("TEST", false)).To(BeFalse())
			Expect(GetBoolEnv("TEST", true)).To(BeFalse())
		})

		It("Config int should get default", func() {
			Expect(GetIntEnv("TEST", 0)).To(BeIdenticalTo(0))
			Expect(GetIntEnv("test", 10)).To(BeIdenticalTo(10))
		})

		It("Config int should get from envs", func() {
			os.Setenv("TEST", "100")
			Expect(GetIntEnv("TEST", 0)).To(BeIdenticalTo(100))
			Expect(GetIntEnv("TEST", 10)).To(BeIdenticalTo(100))
		})
	})
})
