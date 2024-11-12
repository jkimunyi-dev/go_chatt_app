package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

const (
	MessageRate = 1.0

	PORT = "6969"

	SAFE_MODE = false
)

type Client struct {
	Conn        net.Conn
	LastMessage time.Time
}

type MessageType int

const (
	ClientConnected MessageType = iota + 1
	NewMessage
	ClientDisconncted
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
	clients := map[string]*Client{}

	for {
		msg := <-messages

		switch msg.Type {
		case ClientConnected:
			log.Printf("Client Connected  %s", safeRemoteAddress(msg.Conn))
			clients[msg.Conn.RemoteAddr().String()] = &Client{
				Conn:        msg.Conn,
				LastMessage: time.Now(),
			}
		case ClientDisconncted:
			log.Printf("Client Disconnected  %s", safeRemoteAddress(msg.Conn))
			msg.Conn.Close()
			delete(clients, msg.Conn.RemoteAddr().String())
		case NewMessage:
			now := time.Now()

			addr := msg.Conn.RemoteAddr().String()

			author := clients[addr]

			if now.Sub(author.LastMessage).Seconds() >= MessageRate {
				author.LastMessage = now
				log.Printf("Client %s sent message : %s", safeRemoteAddress(msg.Conn), msg.Text)
				for _, client := range clients {
					if client.Conn.RemoteAddr().String() != addr {
						_, err := client.Conn.Write([]byte(msg.Text))
						if err != nil {
							// TODO : Remove connection from the list
							fmt.Printf("Could not send data to : %s : %s\n", safeRemoteAddress(client.Conn), err)
						}
					}
				}
			}

		}
	}
}

func client(conn net.Conn, messages chan Message) {
	// defer conn.Close()

	buffer := make([]byte, 512)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("Could not read from client %s : %s \n", safeRemoteAddress(conn), err)
			conn.Close()
			messages <- Message{
				Type: ClientDisconncted,
				Conn: conn,
			}
			return
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
