package ipc

import (
	"github.com/jc-lab/go-tls-psk"
	"net"
	"sync"
	"time"
)

// Server - holds the details of the server Connection & config.
type Server struct {
	socketDirectory    string
	name               string
	listen             net.Listener
	status             Status
	recieved           chan (*Message)
	maxMsgSize         int
	unMask             int
	securityDescriptor string
	pskConfig          tls.PSKConfig
}

// Connection
type Connection struct {
	conn       net.Conn
	maxMsgSize int
	status     Status
	toWrite    chan (*Message)
	mutex      *sync.Mutex
}

// Client - holds the details of the client Connection and config.
type Client struct {
	socketDirectory string
	name            string
	conn            net.Conn
	status          Status
	timeout         float64       //
	retryTimer      time.Duration // number of seconds before trying to connect again
	recieved        chan (*Message)
	toWrite         chan (*Message)
	maxMsgSize      int
	pskConfig       tls.PSKConfig
}

// Message - contains the  recieved message
type Message struct {
	MsgType    int // type of message sent - 0 is reserved
	Connection *Connection
	err        error  // details of any error
	Data       []byte // message data recieved
	Status     Status
}

// Status - Status of the Connection
type Status int

const (

	// NotConnected - 0
	NotConnected Status = iota
	// Listening - 1
	Listening Status = iota
	// Connecting - 2
	Connecting Status = iota
	// Connected - 3
	Connected Status = iota
	// ReConnecting - 4
	ReConnecting Status = iota
	// Closed - 5
	Closed Status = iota
	// Closing - 6
	Closing Status = iota
	// Error - 7
	Error Status = iota
	// Timeout - 8
	Timeout Status = iota
)

// ServerConfig - used to pass configuation overrides to ServerStart()
type ServerConfig struct {
	SocketDirectory    string
	Timeout            time.Duration
	MaxMsgSize         int
	UseUnmask          bool
	Unmask             int
	SecurityDescriptor string
	PskConfig          tls.PSKConfig
}

// ClientConfig - used to pass configuation overrides to ClientStart()
type ClientConfig struct {
	SocketDirectory string
	Timeout         float64
	RetryTimer      time.Duration
	PskConfig       tls.PSKConfig
}
