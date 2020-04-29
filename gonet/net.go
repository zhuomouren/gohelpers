package gonet

import (
	"errors"
	"fmt"
	"net"
	"time"
)

type GoNet struct{}

var NetHelper = &GoNet{}

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

func (this *GoNet) PrivateIPv4() (net.IP, error) {
	as, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, a := range as {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}

		ip := ipnet.IP.To4()
		if this.IsPrivateIPv4(ip) {
			return ip, nil
		}
	}
	return nil, errors.New("no private ip address")
}

func (this *GoNet) IsPrivateIPv4(ip net.IP) bool {
	return ip != nil &&
		(ip[0] == 10 || ip[0] == 172 && (ip[1] >= 16 && ip[1] < 32) || ip[0] == 192 && ip[1] == 168)
}
