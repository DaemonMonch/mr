package mr

import (
	"testing"
)

func TestR(t *testing.T) {
	a := 1
	b := 2
	l := []int{a, b}
	for _, i := range l {
		t.Logf("%p\n", &i)
		func() {
			t.Logf("f %p\n", &i)
		}()
	}
}
