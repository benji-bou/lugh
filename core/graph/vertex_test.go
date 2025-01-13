package graph_test

import (
	"testing"

	"github.com/benji-bou/lugh/core/graph/graphtest"
)

func TestSynchronizationWorker(t *testing.T) {
	dataTest := [][]byte{[]byte("test"), []byte("toto"), []byte("titi")}

	useSyncWorkerTests := []graphtest.WorkerConfigTest[[]byte]{
		{Name: "Normal Use Sync Worker", DataTest: dataTest, ExpectedOutput: dataTest, IO: graphtest.SingleForwardWorker, Assert: graphtest.AssertDefaultEqualByteSlices},
		{Name: "Without Run Sync Worker", DataTest: dataTest, ExpectedOutput: dataTest, IO: graphtest.NonRunningSingleForwardWorker, Assert: graphtest.AssertShouldNotReceiveData[[]byte]},
	}
	for _, useSyncWorkerTest := range useSyncWorkerTests {
		graphtest.TestWorkerChain(t, useSyncWorkerTest)
	}
}
