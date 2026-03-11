package driver_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WenhanLyu/m1sim/driver"
)

var _ = Describe("Driver", func() {
	It("should have a Driver type", func() {
		var d driver.Driver
		Expect(d).To(BeZero())
	})
})
