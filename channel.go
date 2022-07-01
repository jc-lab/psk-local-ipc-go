package ipc

import "net"

type ServerChannel struct {
	path   string
	addr   net.Addr
	listen net.Listener
}

type ListenConfig struct {
	directory             string
	name                  string
	unixUnmask            int
	winSecurityDescriptor string
}

func (sc *ServerChannel) Addr() net.Addr {
	return sc.addr
}
