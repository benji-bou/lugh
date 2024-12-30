package graph

import (
	"bytes"
	"context"
	"testing"
	"time"
)

type ForwardWorkerTestable[K any] struct {
}

func (s ForwardWorkerTestable[K]) Work(ctx context.Context, input K, yield func(elem K) error) error {
	return yield(input)
}

func startNormalUseSyncWorker() (chan<- []byte, <-chan []byte, <-chan error) {
	inputTestC := make(chan []byte)
	v := NewIOWorkerFromWorker(ForwardWorkerTestable[[]byte]{})
	v.SetInput(inputTestC)
	outputTestC := v.Output()
	ctx := NewContext(context.Background())
	errC := v.Run(ctx) //, NewWorkerSynchronization()
	ctx.Synchronize()
	return inputTestC, outputTestC, errC
}
func startWithoutRunSyncWorker() (chan<- []byte, <-chan []byte, <-chan error) {
	inputTestC := make(chan []byte)
	v := NewIOWorkerFromWorker(ForwardWorkerTestable[[]byte]{})
	v.SetInput(inputTestC)
	outputTestC := v.Output()
	return inputTestC, outputTestC, nil
}

func testUseSyncWorker(t *testing.T, testConfig syncWorkerConfigTest) {
	t.Helper()
	inputTestC, outputTestC, errC := testConfig.f()
	dataTest := testConfig.dataTest
	t.Run(testConfig.name, func(t *testing.T) {
		go func() {
			for _, data := range dataTest {
				inputTestC <- data
			}
			close(inputTestC)
		}()
		testConfig.asserF(t, dataTest, outputTestC, errC)
	})

}

func assertDefaultEqual(t *testing.T, dataTest [][]byte, outputC <-chan []byte, errC <-chan error) {
	t.Helper()
	i := 0
	for {
		select {
		case err := <-errC:
			t.Error(err)
			return
		case outptutData, ok := <-outputC:
			if !ok {
				return
			}
			if !bytes.Equal(outptutData, dataTest[i]) {
				t.Errorf("data not match")
				return
			}
			i++

		}
	}
}

func assertShouldNotReceiveData(t *testing.T, _ [][]byte, outputC <-chan []byte, errC <-chan error) {
	t.Helper()
	tC := time.After(3 * time.Second)

	for {
		select {
		case err := <-errC:
			t.Error(err)
			return
		case _, ok := <-outputC:
			if ok {
				t.Errorf("received data but should not")
				return
			}
		case <-tC:
			return

		}
	}
}

type syncWorkerConfigTest struct {
	f        func() (chan<- []byte, <-chan []byte, <-chan error)
	name     string
	dataTest [][]byte
	asserF   func(t *testing.T, dataTest [][]byte, outputC <-chan []byte, errC <-chan error)
}

func TestSyncWorker(t *testing.T) {
	dataTest := [][]byte{[]byte("test"), []byte("toto"), []byte("titi")}

	useSyncWorkerTests := []syncWorkerConfigTest{
		{startNormalUseSyncWorker, "Normal Use Sync Worker", dataTest, assertDefaultEqual},
		{startWithoutRunSyncWorker, "Without Run Sync Worker", dataTest, assertShouldNotReceiveData},
	}

	for _, useSyncWorkerTest := range useSyncWorkerTests {
		testUseSyncWorker(t, useSyncWorkerTest)
	}
}
