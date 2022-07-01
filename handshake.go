package ipc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// 1st message sent from the server
// byte 0 = protocal version no.
// byte 1 = whether encryption is to be used - 0 no , 1 = encryption
func (sc *Server) handshake(connection *Connection) error {
	buff := make([]byte, 8)

	buff[0] = byte(version)
	binary.BigEndian.PutUint32(buff[4:], maxMsgSize)

	_, err := connection.conn.Write(buff)
	if err != nil {
		return errors.New("unable to send handshake: " + err.Error())
	}

	recv := make([]byte, 1)
	_, err = connection.conn.Read(recv)
	if err != nil {
		return errors.New("failed to recieve handshake reply")
	}

	switch result := recv[0]; result {
	case 0:
		return nil
	case 1:
		return errors.New("client has a different version number")
	}

	return errors.New("other error - handshake failed")
}

// 1st message recieved by the client
func (cc *Client) handshake() error {
	recv := make([]byte, 8)
	_, err := cc.conn.Read(recv)
	if err != nil {
		return errors.New("failed to recieve handshake message: " + err.Error())
	}

	if recv[0] != version {
		cc.handshakeSendReply(1)
		return errors.New(fmt.Sprintf("server has sent a different version number: %d", int(recv[0]&0xff)))
	}

	var maxMsgSize uint32
	binary.Read(bytes.NewReader(recv[4:]), binary.BigEndian, &maxMsgSize)
	cc.maxMsgSize = int(maxMsgSize)

	cc.handshakeSendReply(0) // 0 is ok

	return nil
}

func (cc *Client) handshakeSendReply(result byte) {
	buff := make([]byte, 1)
	buff[0] = result

	cc.conn.Write(buff)
}
