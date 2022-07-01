package main

import (
	"errors"
	"fmt"
	"github.com/jc-lab/go-tls-psk"
	ipc "github.com/jc-lab/psk-local-ipc/go"
	"log"
	"time"
)

var defaultPskConfig = tls.PSKConfig{
	GetIdentity: func() string {
		return "hello"
	},
	GetKey: func(identity string) ([]byte, error) {
		if identity == "hello" {
			return []byte("world"), nil
		}
		return nil, errors.New("INVALID IDENTITY: " + identity)
	},
}
var defaultServerConfig = &ipc.ServerConfig{PskConfig: defaultPskConfig}
var defaultClientConfig = &ipc.ClientConfig{PskConfig: defaultPskConfig}

func main() {
	log.Println("starting")

	server()

	//client()
}

func server() {
	sc, err := ipc.StartServer("testtest2", defaultServerConfig)
	if err != nil {
		log.Println(err)
		return
	}

	for {
		m, err := sc.Read()

		if err == nil {
			if m.Status == ipc.Connected {
				println("CONNECTED: ", m)
				go serverSend(m.Connection)
				//go serverSend1(connection)
				//serverSend2(connection)

			} else if m.MsgType > 0 {
				log.Println("Server recieved: "+string(m.Data)+" - Message type: ", m.MsgType)
			} else {
				log.Println("Server event: " + m.Status.String())
			}

		} else {
			log.Println("Server error")
			log.Println(err)
			//break
		}
	}
}

func serverSend(connection *ipc.Connection) {

	for {

		err := connection.Write(3, []byte("Hello Client 4"))

		if err != nil {
			break
			//fmt.Println(err)
		}

		time.Sleep(time.Second / 30)

	}
}

func serverSend1(connection *ipc.Connection) {

	for {

		connection.Write(5, []byte("Hello Client 1"))
		connection.Write(7, []byte("Hello Client 2"))
		connection.Write(9, []byte("Hello Client 3"))

		time.Sleep(time.Second / 30)

	}

}

func serverSend2(connection *ipc.Connection) {

	for {

		err := connection.Write(88, []byte("Hello Client 7"))
		err = connection.Write(99, []byte("Hello Client 8"))
		err = connection.Write(22, []byte("Hello Client 9"))

		if err != nil {
			//fmt.Println(err)
		}

		time.Sleep(time.Second / 30)

	}
}

func client() {
	cc, err := ipc.StartClient("testtest2", defaultClientConfig)
	if err != nil {
		log.Println(err)
		return
	}

	go func() {
		for {
			m, err := cc.Read()

			if err != nil {
				// An error is only returned if the recieved channel has been closed,
				//so you know the connection has either been intentionally closed or has timmed out waiting to connect/re-connect.
				break
			}

			fmt.Printf("cc.Status: %d: %s\n", m.MsgType, m.Status.String())

			if m.MsgType == -1 { // message type -1 is status change
				//log.Println("Status: " + m.Status)
			}

			if m.MsgType == -2 { // message type -2 is an error, these won't automatically cause the recieve channel to close.
				log.Println("Error: " + err.Error())
			}

			if m.MsgType > 0 { // all message types above 0 have been recieved over the connection

				log.Println(" Message type: ", m.MsgType)
				log.Println("Client recieved: " + string(m.Data))
			}
			//}
		}

	}()

	clientSend2(cc)
}

func clientSend(cc *ipc.Client) {

	for {

		_ = cc.Write(14, []byte("hello server 4"))
		_ = cc.Write(44, []byte("hello server 5"))
		_ = cc.Write(88, []byte("hello server 6"))

		time.Sleep(time.Second / 20)

	}

}

func clientSend1(cc *ipc.Client) {

	for {

		_ = cc.Write(1, []byte("hello server 1"))
		_ = cc.Write(9, []byte("hello server 2"))
		_ = cc.Write(34, []byte("hello server 3"))

		time.Sleep(time.Second / 20)

	}

}

func clientSend2(cc *ipc.Client) {

	for {

		_ = cc.Write(444, []byte("hello server 7"))
		_ = cc.Write(234, []byte("hello server 8"))
		_ = cc.Write(111, []byte("hello server 9"))

		time.Sleep(time.Second / 20)

	}
}

func clientRecv(c *ipc.Client) {

	for {
		m, err := c.Read()

		if err != nil {
			// An error is only returned if the recieved channel has been closed,
			//so you know the connection has either been intentionally closed or has timmed out waiting to connect/re-connect.
			break
		}

		if m.MsgType == -1 { // message type -1 is status change
			//log.Println("Status: " + m.Status)
		}

		if m.MsgType == -2 { // message type -2 is an error, these won't automatically cause the recieve channel to close.
			log.Println("Error: " + err.Error())
		}

		if m.MsgType > 0 { // all message types above 0 have been recieved over the connection

			log.Println(" Message type: ", m.MsgType)
			log.Println("Client recieved: " + string(m.Data))
		}
		//}
	}

}
