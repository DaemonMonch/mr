package mr

import (
	"context"
	"testing"
	"time"
)

func TestTcpCheck(t *testing.T) {
	c := NewTcpChecker(context.Background(), HealthCheck{Address: "localhost:42301", MaxAwaitTime: 1 * time.Second})
	err := c.Check()
	if err != nil {
		t.Error(err)
	}
}
