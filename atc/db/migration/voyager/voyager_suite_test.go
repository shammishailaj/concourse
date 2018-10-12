package voyager_test

import (
	"os"
	"time"

	"github.com/concourse/concourse/atc/postgresrunner"
	"github.com/gobuffalo/packr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"

	"testing"
)

func TestVoyager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Voyager Suite")
}

var postgresRunner postgresrunner.Runner
var dbProcess ifrit.Process

var _ = BeforeSuite(func() {
	postgresRunner = postgresrunner.Runner{
		Port: 5433 + GinkgoParallelNode(),
	}
	dbProcess = ifrit.Invoke(postgresRunner)
})

var _ = BeforeEach(func() {
	postgresRunner.CreateTestDB()
})

var _ = AfterEach(func() {
	postgresRunner.DropTestDB()
})

var _ = AfterSuite(func() {
	dbProcess.Signal(os.Interrupt)
	Eventually(dbProcess.Wait(), 10*time.Second).Should(Receive())
})

var asset = packr.NewBox("./migrations").MustBytes
