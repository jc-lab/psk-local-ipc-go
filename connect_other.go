//go:build linux || darwin
// +build linux darwin

package ipc

import (
	"errors"
	"net"
	"os"
	"strings"
	"syscall"
	"time"
)

func buildPipePath(prefix string, name string) string {
	base := prefix
	if base == "" {
		base = "/tmp/"
	} else if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return base + name + ".sock"
}

// Server create a unix socket and start listening connections - for unix and linux
func (sc *Server) createListenSocket() (net.Listener, error) {
	sockPath := buildPipePath(sc.socketDirectory, sc.name)

	if err := os.RemoveAll(sockPath); err != nil {
		return nil, err
	}

	var oldUmask int
	if sc.unMask >= 0 {
		oldUmask = syscall.Umask(sc.unMask)
	}

	listen, err := net.Listen("unix", sockPath)

	if sc.unMask >= 0 {
		syscall.Umask(oldUmask)
	}

	if err != nil {
		return nil, err
	}

	return listen, nil
}

// Client connect to the unix socket created by the server -  for unix and linux
func (cc *Client) dial() (net.Conn, error) {
	pipePath := buildPipePath(cc.socketDirectory, cc.name)

	startTime := time.Now()

	for {
		if cc.timeout != 0 {
			if time.Now().Sub(startTime).Seconds() > cc.timeout {
				cc.status = Closed
				return nil, errors.New("Timed out trying to connect")
			}
		}

		conn, err := net.Dial("unix", pipePath)
		if err == nil {
			return conn, nil
		} else {
			if strings.Contains(err.Error(), "connect: no such file or directory") == true {

			} else if strings.Contains(err.Error(), "connect: Connection refused") == true {

			} else {
				cc.recieved <- &Message{err: err, MsgType: -2}
			}
		}

		time.Sleep(cc.retryTimer * time.Second)

	}

}
