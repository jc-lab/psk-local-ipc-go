package ipc

import (
	"bufio"
	"errors"
	"github.com/jc-lab/go-tls-psk"
	"strings"
	"time"
)

// StartClient - start the ipc client.
//
// ipcName = is the name of the unix socket or named pipe that the client will try and connect to.
// timeout = number of seconds before the socket/pipe times out trying to connect/re-cconnect - if -1 or 0 it never times out.
// retryTimer = number of seconds before the client tries to connect again.
//
func StartClient(ipcName string, config *ClientConfig) (*Client, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	err := checkIpcName(ipcName)
	if err != nil {
		return nil, err
	}

	cc := &Client{
		socketDirectory: config.SocketDirectory,
		name:            ipcName,
		status:          NotConnected,
		recieved:        make(chan *Message),
		toWrite:         make(chan *Message),
		pskConfig:       config.PskConfig,
	}

	if config == nil {
		cc.timeout = 0
		cc.retryTimer = time.Duration(1)
	} else {
		if config.Timeout < 0 {
			cc.timeout = 0
		} else {
			cc.timeout = config.Timeout
		}

		if config.RetryTimer < 1 {
			cc.retryTimer = time.Duration(1)
		} else {
			cc.retryTimer = time.Duration(config.RetryTimer)
		}
	}

	go startClient(cc)

	return cc, nil

}

func startClient(cc *Client) {
	cc.status = Connecting
	cc.recieved <- &Message{Status: cc.status, MsgType: -1}

	err := cc.createConnection()
	if err != nil {
		cc.recieved <- &Message{err: err, MsgType: -2}
		return
	}

	go cc.read()
	go cc.write()
}

func (cc *Client) read() {
	bLen := make([]byte, 4)

	for {
		res := cc.readData(bLen)
		if res == false {
			break
		}

		mLen := bytesToInt(bLen)

		msgRecvd := make([]byte, mLen)

		res = cc.readData(msgRecvd)
		if res == false {
			break
		}

		if bytesToInt(msgRecvd[:4]) == 0 {
			//  type 0 = control message
		} else {
			cc.recieved <- &Message{Status: cc.status, Data: msgRecvd[4:], MsgType: bytesToInt(msgRecvd[:4])}
		}
	}
}

func (cc *Client) readData(buff []byte) bool {
	_, err := cc.conn.Read(buff)
	if err != nil {
		if strings.Contains(err.Error(), "EOF") { // the Connection has been closed by the client.
			cc.conn.Close()

			if cc.status != Closing || cc.status == Closed {
				go cc.reconnect()
			}
			return false
		}

		if cc.status == Closing {
			cc.status = Closed
			cc.recieved <- &Message{Status: cc.status, MsgType: -1}
			cc.recieved <- &Message{err: errors.New("Client has closed the Connection"), MsgType: -2}
			return false
		}

		// other read error
		return false

	}

	return true
}

func (cc *Client) reconnect() {
	cc.status = ReConnecting
	cc.recieved <- &Message{Status: cc.status, MsgType: -1}

	err := cc.createConnection() // connect to the pipe
	if err != nil {
		if err.Error() == "Timed out trying to connect" {
			cc.status = Timeout
			cc.recieved <- &Message{Status: cc.status, MsgType: -1}
			cc.recieved <- &Message{err: errors.New("Timed out trying to re-connect"), MsgType: -2}
		}

		return
	}

	go cc.read()
}

func (cc *Client) createConnection() error {
	conn, err := cc.dial() // connect to the pipe
	if err != nil {
		return err
	}
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS10,
		MaxVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_PSK_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_PSK_WITH_AES_256_CBC_SHA384,
			tls.TLS_ECDHE_PSK_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_PSK_WITH_AES_128_CBC_SHA,
		},
		InsecureSkipVerify: true,
		Extra:              cc.pskConfig,
	}
	cc.conn = tls.Client(conn, tlsConfig)

	err = cc.handshake()
	if err != nil {
		return err
	}

	cc.status = Connected
	cc.recieved <- &Message{Status: cc.status, MsgType: -1}

	return nil
}

// Read - blocking function that waits until an non multipart message is recieved
// returns the message type, data and any error.
//
func (cc *Client) Read() (*Message, error) {
	m, ok := (<-cc.recieved)
	if ok == false {
		return nil, errors.New("the recieve channel has been closed")
	}

	if m.err != nil {
		close(cc.recieved)
		close(cc.toWrite)
		return nil, m.err
	}

	return m, nil
}

// Write - writes a non multipart message to the ipc Connection.
// msgType - denotes the type of data being sent. 0 is a reserved type for internal messages and errors.
//
func (cc *Client) Write(msgType int, message []byte) error {

	if msgType == 0 {
		return errors.New("Message type 0 is reserved")
	}

	if cc.status != Connected {
		return errors.New(cc.status.String())
	}

	mlen := len(message)
	if mlen > cc.maxMsgSize {
		return errors.New("Message exceeds maximum message length")
	}

	cc.toWrite <- &Message{MsgType: msgType, Data: message}

	return nil

}

func (cc *Client) write() {
	for {
		m, ok := <-cc.toWrite

		if ok == false {
			break
		}

		toSend := intToBytes(m.MsgType)

		writer := bufio.NewWriter(cc.conn)

		toSend = append(toSend, m.Data...)

		writer.Write(intToBytes(len(toSend)))
		writer.Write(toSend)

		err := writer.Flush()
		if err != nil {
			//return err
		}
	}
}

// Status - returns the current Connection status as a string
func (cc *Client) Status() Status {
	return cc.status
}

// Close - closes the Connection
func (cc *Client) Close() {

	cc.status = Closing
	cc.conn.Close()
}
