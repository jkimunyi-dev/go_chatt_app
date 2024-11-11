package main

import (
	"fmt"
	"log"
	"net"
)

const PORT = "6969"

const SAFE_MODE = true

type Client struct {
	conn     net.Conn
	outgoing chan string
}

type MessageType int

const (
	ClientConnected MessageType = iota + 1
	NewMessage
	DeleteClient
)

type Message struct {
	Type MessageType
	Conn net.Conn
	Text string
}

func safeRemoteAddress(conn net.Conn) string {
	if SAFE_MODE {
		return "[REDACTED]"
	} else {
		return conn.RemoteAddr().String()
	}
}

func server(messages chan Message) {
	conns := make([]net.Conn, 512)

	for {
		msg := <-messages

		switch msg.Type {
		case ClientConnected:
			conns = append(conns, msg.Conn)
		case DeleteClient:
			msg.Conn.Close()
		case NewMessage:
			for _, conn := range conns {
				_, err := conn.Write([]byte(msg.Text))
				if err != nil {
					// TODO : Remove connection from the list
					fmt.Printf("Could not send data to : %s : %s", safeRemoteAddress((conn)), err)
				}
			}
		}
	}
}

func client(conn net.Conn, messages chan Message) {

	buffer := make([]byte, 512)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			messages <- Message{
				Type: DeleteClient,
				Conn: conn,
			}
		}

		messages <- Message{
			Type: NewMessage,
			Text: string(buffer[0:n]),
			Conn: conn,
		}

	}
}

func main() {
	ln, err := net.Listen("tcp", ":"+PORT)
	if err != nil {
		log.Fatalf("Could not connect to port : %s : %s \n", PORT, err)
	}

	log.Printf("Listening to TCP connection on port : %s ...\n", PORT)

	messages := make(chan Message)

	go server(messages)

	for {
		conn, err := ln.Accept()

		if err != nil {
			log.Printf("Connection not accepted : %s : %s \n", err)
		}

		fmt.Printf("Accepted connection from : %s \n", safeRemoteAddress(conn))

		messages <- Message{
			Type: ClientConnected,
			Conn: conn,
		}

		go client(conn, messages)
	}

}
