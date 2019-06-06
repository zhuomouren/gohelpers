package gonet

import (
	"fmt"
	"net"
	"time"
)

type GoNet struct{}

var Helper = &GoNet{}

// Ping connects to the address on the named network,
// using net.DialTimeout, and immediately closes it.
// It returns the connection error. A nil value means success.
// For examples of valid values of network and address,
// see the documentation of net.Dial
// err := Ping("tcp", "0.0.0.0:8088", time.Second)
func (this *GoNet) Ping(network, address string, timeout time.Duration) error {
	conn, err := net.DialTimeout(network, address, timeout)
	if conn != nil {
		defer conn.Close()
	}
	return err
}

// 检查端口是否可用
func (this *GoNet) PortIsAvailable(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("0.0.0.0:%d", port), time.Second)
	if conn != nil {
		defer conn.Close()
	}

	return err == nil
}
