package utils

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestGetFirstIpServiceCidr(t *testing.T) {
	g := NewWithT(t)

	t.Run("get second ip in the cidr", func(t *testing.T) {
		ip := getFirstIpServiceCidr("192.169.0.0/16")
		g.Expect(ip).To(Equal("192.169.0.1"))
	})
}
