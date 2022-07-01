package ipc

import (
	"bufio"
	"errors"
	"github.com/jc-lab/go-tls-psk"
	"sync"
	"time"
)

// StartServer - starts the ipc server.
//
// ipcName = is the name of the unix socket or named pipe that will be created.
// timeout = number of seconds before the socket/pipe times out waiting for a Connection/re-cconnection - if -1 or 0 it never times out.
//
func StartServer(ipcName string, config *ServerConfig) (*Server, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	err := checkIpcName(ipcName)
	if err != nil {
		return nil, err
	}

	sc := &Server{
		name:               ipcName,
		status:             NotConnected,
		recieved:           make(chan *Message),
		unMask:             -1,
		pskConfig:          config.PskConfig,
		socketDirectory:    config.SocketDirectory,
		securityDescriptor: config.SecurityDescriptor,
	}

	if config.MaxMsgSize < 1024 {
		sc.maxMsgSize = maxMsgSize
	} else {
		sc.maxMsgSize = config.MaxMsgSize
	}

	if config.UseUnmask {
		sc.unMask = config.Unmask
	}

	go startServer(sc)

	return sc, err
}

func startServer(sc *Server) {
	listen, err := sc.createListenSocket()
	if err != nil {
		sc.recieved <- &Message{err: err, MsgType: -2}
		return
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
		Extra:              sc.pskConfig,
		Certificates:       []tls.Certificate{tls.Certificate{}},
	}
	sc.listen = tls.NewListener(listen, tlsConfig)
	sc.status = Listening

	go sc.acceptLoop()

	sc.recieved <- &Message{Status: sc.status, MsgType: -1}
}

func (sc *Server) acceptLoop() {
	for {
		conn, err := sc.listen.Accept()
		if err != nil {
			break
		}

		connection := &Connection{
			maxMsgSize: sc.maxMsgSize,
			conn:       conn,
			status:     Connecting,
			toWrite:    make(chan *Message),
			mutex:      &sync.Mutex{},
		}

		sc.recieved <- &Message{
			MsgType:    -2,
			Connection: connection,
			Status:     connection.status,
		}

		err2 := sc.handshake(connection)
		if err2 != nil {
			sc.recieved <- &Message{err: err2, MsgType: -2}
			conn.Close()
		} else {
			go sc.read(connection)
			go sc.write(connection)

			connection.status = Connected

			sc.recieved <- &Message{
				MsgType:    -2,
				Connection: connection,
				Status:     connection.status,
			}
		}
	}

}

func (sc *Server) read(connection *Connection) {
	bLen := make([]byte, 4)

	for {

		res := sc.readData(connection, bLen)
		if res == false {
			break
		}

		mLen := bytesToInt(bLen)

		msgRecvd := make([]byte, mLen)

		res = sc.readData(connection, msgRecvd)
		if res == false {
			break
		}

		if bytesToInt(msgRecvd[:4]) == 0 {
			//  type 0 = control message
		} else {
			if sc.Status() != Closed {
				sc.recieved <- &Message{Connection: connection, Data: msgRecvd[4:], MsgType: bytesToInt(msgRecvd[:4])}
			}
		}
	}
}

func (sc *Server) readData(connection *Connection, buff []byte) bool {
	_, err := connection.conn.Read(buff)
	if err != nil {

		oldStatus := connection.status
		connection.status = Closed

		if sc.Status() != Closed {
			sc.recieved <- &Message{Connection: connection, Status: connection.status, MsgType: -1}
		}

		if oldStatus == Closing {
			return false
		}

		connection.Close()

		return false
	}

	return true
}

// Read - blocking function that waits until an non multipart message is recieved

func (sc *Server) Read() (*Message, error) {

	m, ok := (<-sc.recieved)
	if ok == false {
		return nil, errors.New("the recieve channel has been closed")
	}

	if m.err != nil {
		return nil, m.err
	}

	return m, nil

}

// Write - writes a non multipart message to the ipc Connection.
// msgType - denotes the type of data being sent. 0 is a reserved type for internal messages and errors.
//
func (connection *Connection) Write(msgType int, message []byte) error {

	if msgType == 0 {
		return errors.New("Message type 0 is reserved")
	}

	mlen := len(message)

	if mlen > connection.maxMsgSize {
		return errors.New("Message exceeds maximum message length")
	}

	connection.mutex.Lock()
	if connection.status == Connected {
		connection.toWrite <- &Message{MsgType: msgType, Data: message}
		connection.mutex.Unlock()
	} else {
		connection.mutex.Unlock()
		return errors.New(connection.status.String())
	}

	return nil

}

func (sc *Server) write(connection *Connection) {

	for {

		m, ok := <-connection.toWrite

		if ok == false {
			break
		}

		toSend := intToBytes(m.MsgType)

		writer := bufio.NewWriter(connection.conn)

		toSend = append(toSend, m.Data...)

		writer.Write(intToBytes(len(toSend)))
		writer.Write(toSend)

		err := writer.Flush()
		if err != nil {
			//return err
		}

		time.Sleep(2 * time.Millisecond)

	}
}

// Status - returns the current Connection status as a string
func (sc *Server) Status() Status {
	return sc.status
}

// Close - closes the Connection
func (sc *Server) Close() {

	sc.status = Closed
	sc.listen.Close()
	close(sc.recieved)

}

func (connection *Connection) Close() {
	connection.mutex.Lock()
	if connection.status == Closed {
		close(connection.toWrite)
	} else {
		connection.status = Closing
		connection.conn.Close()
	}
	connection.mutex.Unlock()
}
