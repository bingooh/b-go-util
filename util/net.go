package util

import (
	"fmt"
	"net"
)

func IsTcpAddressInUse(address string) bool {
	ln, err := net.Listen(`tcp`, address)
	if err == nil {
		_ = ln.Close()
		return false
	}

	return true
}

func IsTcpPortInUse(port int) bool {
	//以下未判断局域网IP，可以同时监听同1端口，不同本机IP的地址
	return IsTcpAddressInUse(fmt.Sprintf(`:%v`, port)) ||
		IsTcpAddressInUse(fmt.Sprintf(`127.0.0.1:%v`, port))
}

func NextUnusedTcpPort(start int) int {
	for i := start; i < 65535; i++ {
		if !IsTcpPortInUse(i) {
			return i
		}
	}

	return -1
}
