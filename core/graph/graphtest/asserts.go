package graphtest

import (
	"bytes"
	"testing"
	"time"
)

// assertDefaultEqual is an assert helper function that listen the outputChan
// and ensure that the output is equal to the dataTest in the correct order
// it err is received on the chan it will fail the test and return
func AssertDefaultEqual[K comparable](t *testing.T, dataTest []K, outputC <-chan K, errC <-chan error) {
	t.Helper()
	i := 0
	for {
		select {
		case err, ok := <-errC:
			if !ok && i != len(dataTest) {
				t.Errorf("errC closed without getting al data")
				return
			}
			if err != nil {
				t.Errorf("AssertDefaultEqual received an error: %s", err.Error())
			}
			return
		case outptutData, ok := <-outputC:
			if !ok {
				if i != len(dataTest) {
					t.Errorf("AssertDefaultEqual number of data received not match. Want %d got %d", len(dataTest), i)
				}
				return
			}
			if outptutData != dataTest[i] {
				t.Errorf("AssertDefaultEqual data not match, want %v got %v", dataTest[i], outptutData)
			}
			i++
		}
	}
}

func AssertDefaultEqualByteSlices(t *testing.T, dataTest [][]byte, outputC <-chan []byte, errC <-chan error) {
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

// assertShouldNotReceiveData is an assert helper function that listen the outputChan
// the assert failed if data is received on the chan within 3 seconds.
func AssertShouldNotReceiveData[K any](t *testing.T, _ []K, outputC <-chan K, errC <-chan error) {
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
