//go:build windows
// +build windows

package ipc

import (
	"errors"
	"net"
	"strings"
	"time"

	"github.com/Microsoft/go-winio"
)

func buildPipePath(prefix string, name string) string {
	base := prefix
	if base == "" {
		base = `\\.\pipe\`
	} else if !strings.HasSuffix(base, `\`) {
		base += `\`
	}
	return base + name
}

// Server function
// Create the named pipe (if it doesn't already exist) and start listening for a client to connect.
// when a client connects and Connection is accepted the read function is called on a go routine.
func (sc *Server) createListenSocket() (net.Listener, error) {
	pipePath := buildPipePath(sc.socketDirectory, sc.name)

	pipeConfig := &winio.PipeConfig{}

	if sc.securityDescriptor != "" {
		pipeConfig.SecurityDescriptor = sc.securityDescriptor
	}

	listen, err := winio.ListenPipe(pipePath, pipeConfig)
	if err != nil {
		return nil, err
	}

	return listen, nil
}

// Client function
// dial - attempts to connect to a named pipe created by the server
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
		pn, err := winio.DialPipe(pipePath, nil)
		if err == nil {
			return pn, nil
		} else {
			if !strings.Contains(err.Error(), "The system cannot find the file specified.") {
				return nil, err
			}
		}

		time.Sleep(cc.retryTimer * time.Second)
	}
}
