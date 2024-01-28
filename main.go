package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"reflect"
	"strings"
)

type Server struct {
	listenAddr string
	ln         net.Listener
	quitch     chan struct{}
	msgch      chan Message
}

type Message struct {
	from    string
	payload string
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		msgch:      make(chan Message, 10),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()
	s.ln = ln

	go s.acceptLoop()

	<-s.quitch
	close(s.msgch)

	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("Accept error")
			continue
		}
		fmt.Println("New connection: ", conn.RemoteAddr())
		go s.handleClient(conn)
	}
}


func (s *Server) handleClient(conn net.Conn) {
	defer conn.Close()

	// Prompt the client for their name
	s.sendMessage(conn, "Please enter your name: ")
	name, err := s.receiveMessage(conn)

	if err != nil {
		fmt.Println("Error receiving name:", err)
		return
	}

	clientName := strings.TrimSpace(name)
	s.sendMessage(conn, "Welcome to TCP-Chat!")

	for {
		// Read the message from the client
		message, err := s.receiveMessage(conn)
		if err != nil {
			fmt.Printf("%s has left the chat\n", clientName)
			break
		}

		s.msgch <- Message{
			from:    clientName,
			payload: message,
		}
	}
}

func (s *Server) sendMessage(conn net.Conn, message string) {
	conn.Write([]byte(message + "\n"))
}

func (s *Server) receiveMessage(conn net.Conn) (string, error) {
	message, err := bufio.NewReader(conn).ReadString('\n')
	return strings.TrimSpace(message), err
}

func main() {
	args := os.Args[1:]
	if len(args) <= 1 {
		port := "8989"
		if len(args) == 1 {
			if !(IsInt(args[0])) {
				fmt.Println("[USAGE]: ./TCPChat $port")
				os.Exit(0)
			}
			port = args[0]
		}
		server := NewServer(":" + port)
		fmt.Println("Listening on the port:", port)
		go func() {
			for msg := range server.msgch {
				fmt.Printf("Message from %s: %s\n", msg.from, msg.payload)
			}
		}()
		server.Start()
	} else {
		fmt.Println("[USAGE]: ./TCPChat $port")
		os.Exit(0)
	}
}

func IsInt(value interface{}) bool {
	kind := reflect.TypeOf(value).Kind()

	return kind == reflect.Int
}
