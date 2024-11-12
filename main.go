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

	SAFE_MODE = true

	BanLimit = 10.0

	StrikeLimit = 10
)

type Client struct {
	Conn        net.Conn
	LastMessage time.Time
	StrikeCount int
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

func sensitive(message string) string {
	if SAFE_MODE {
		return "[REDACTED]"
	} else {
		return message
	}
}

func server(messages chan Message) {
	clients := map[string]*Client{}

	bannedMfs := map[string]time.Time{}

	for {
		msg := <-messages

		switch msg.Type {
		case ClientConnected:
			addr := msg.Conn.RemoteAddr().(*net.TCPAddr)

			bannedAt, banned := bannedMfs[addr.IP.String()]

			now := time.Now()

			if banned {
				if now.Sub(bannedAt).Seconds() > BanLimit {
					delete(bannedMfs, addr.IP.String())
					banned = false
				}
			}

			if !banned {
				log.Printf("Client Connected  %s", sensitive(addr.String()))
				clients[msg.Conn.RemoteAddr().String()] = &Client{
					Conn:        msg.Conn,
					LastMessage: time.Now(),
				}
			} else {
				msg.Conn.Write([]byte("You are banned \n"))
				msg.Conn.Close()
			}

		case ClientDisconncted:
			addr := msg.Conn.RemoteAddr().(*net.TCPAddr)
			log.Printf("Client Disconnected  %s", sensitive(addr.String()))
			msg.Conn.Close()
			delete(clients, msg.Conn.RemoteAddr().String())

		case NewMessage:
			now := time.Now()

			authorAddr := msg.Conn.RemoteAddr().(*net.TCPAddr)

			author := clients[authorAddr.String()]

			if now.Sub(author.LastMessage).Seconds() >= MessageRate {
				author.StrikeCount = 0

				author.LastMessage = now

				log.Printf("Client %s sent message : %s", sensitive(authorAddr.String()), msg.Text)
				for _, client := range clients {
					if client.Conn.RemoteAddr().String() != client.Conn.RemoteAddr().String() {
						_, err := client.Conn.Write([]byte(msg.Text))
						if err != nil {
							// TODO : Remove connection from the list
							fmt.Printf("Could not send data to : %s : %s\n", sensitive(authorAddr.String()), err)
						}
					}
				}

			} else {
				author.StrikeCount += 1
				if author.StrikeCount >= StrikeLimit {
					// Bann user
					bannedMfs[authorAddr.IP.String()] = now
					author.Conn.Write([]byte("You are banned \n"))
					msg.Conn.Close()
				}
			}

		}
	}
}

func client(conn net.Conn, messages chan Message) {
	// defer conn.Close()

	buffer := make([]byte, 512)

	addr := conn.RemoteAddr().(*net.TCPAddr)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("Could not read from client %s : %s \n", sensitive(addr.String()), err)
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

		addr := conn.RemoteAddr().(*net.TCPAddr)

		if err != nil {
			log.Printf("Connection not accepted : %s : %s \n", err)
		}

		fmt.Printf("Accepted connection from : %s \n", sensitive(addr.String()))

		messages <- Message{
			Type: ClientConnected,
			Conn: conn,
		}

		go client(conn, messages)
	}

}
