package mem_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WenhanLyu/m1sim/timing/mem"
)

var _ = Describe("Mem", func() {
	It("should have a MemoryController type", func() {
		var m mem.MemoryController
		Expect(m).To(BeZero())
	})
})
