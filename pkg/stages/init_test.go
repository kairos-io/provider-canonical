package stages

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestAppendIfNotPresent(t *testing.T) {
	g := NewWithT(t)

	t.Run("appends element when not present", func(t *testing.T) {
		slice := []string{"a", "b"}
		result := appendIfNotPresent(slice, "c")
		g.Expect(result).To(Equal([]string{"a", "b", "c"}))
	})

	t.Run("not append when element already present", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		result := appendIfNotPresent(slice, "b")
		g.Expect(result).To(Equal([]string{"a", "b", "c"}))
	})
}
