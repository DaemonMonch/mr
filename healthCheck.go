package mr

import (
	"fmt"
	"io"
	"net"
	"time"
)

type HealthChecker interface {
	Check() error
}

func GetHealthChecker(config *HealthCheck) HealthChecker {
	if config == nil {
		return &NoopChecker{}
	}
	return NewTcpChecker(config)
}

type NoopChecker struct{}

func (tc *NoopChecker) Check() error {
	return nil
}

type TcpChecker struct {
	config        *HealthCheck
	dialer        *net.Dialer
	conn          net.Conn
	b             []byte
	connAliveTime time.Duration
}

func NewTcpChecker(config *HealthCheck) *TcpChecker {
	dialer := &net.Dialer{
		Timeout: config.MaxAwaitTime,
	}
	return &TcpChecker{config: config, dialer: dialer, b: make([]byte, 1)}
}

func (tc *TcpChecker) Check() error {
	fmt.Printf("health check %s \n", tc.config.Address)
	if tc.conn == nil {
		if err := tc.connect(); err != nil {
			return err
		}
	}
	tc.conn.SetDeadline(time.Now().Add(tc.config.MaxAwaitTime))
	_, err := tc.conn.Read(tc.b)
	if io.EOF == err {
		if err := tc.connect(); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TcpChecker) connect() error {
	conn, err := tc.dialer.Dial("tcp", tc.config.Address)
	if err != nil {
		return err
	}
	tc.conn = conn
	return nil
}
