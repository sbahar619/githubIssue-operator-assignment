package auth

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auth Suite")
}

var _ = Describe("GetGitHubToken", func() {
	const testToken = "test_token"
	AfterEach(func() {
		Expect(os.Unsetenv(GitHubTokenEnvVar)).To(Succeed())
	})

	It("should return token when set", func() {
		Expect(os.Setenv(GitHubTokenEnvVar, testToken)).To(Succeed())
		token, err := GetGitHubToken()
		Expect(err).NotTo(HaveOccurred())
		Expect(token).To(Equal(testToken))
	})

	It("should error when not set", func() {
		_, err := GetGitHubToken()
		Expect(err).To(HaveOccurred())
	})
})
