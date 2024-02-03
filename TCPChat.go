package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type Server struct {
	listenAddr      string
	ln              net.Listener
	quitch          chan struct{}
	msgch           chan Message
	clients         map[net.Conn]string // Added a map to store connected clients and their names
	mutex           sync.Mutex
	connectionCount int
	messageBuffer   []Message
}

type Message struct {
	from    string
	payload string
	time    string
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		msgch:      make(chan Message, 10),
		clients:    make(map[net.Conn]string),
	}
}

var logo = `           _nnnn_
          dGGGGMMb
         @p~qp~~qMb
         M|@||@) M|
         @,----.JM|
        JS^\__/  qKL
       dZP        qKRb
      dZP          qKKb
     fZP            SMMb
     HZM            MMMM
     FqM            MMMM
   __| ".        |\dS"qML
   |    '.       | '' \Zq
  _)      \.___.,|     .'
  \____   )MMMMMP|   .'
       '-'       '--'`


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

func (s *Server) sendMessagesToClient(conn net.Conn) {
	for _, msg := range s.messageBuffer {
		s.sendMessage(conn, fmt.Sprintf("[%s][%s]:%s\n", msg.time, msg.from, msg.payload))
	}
}

func (s *Server) acceptLoop() {
	for {
		// Lock the mutex to safely check and update the connection count
		s.mutex.Lock()

		// Check if the maximum connection limit has been reached
		if s.connectionCount >= 10 {
			fmt.Println("Maximum connection limit reached. Not accepting more connections.")

			// Unlock the mutex before breaking out of the loop
			s.mutex.Unlock()
			break
		}

		// Unlock the mutex to allow accepting a new connection
		s.mutex.Unlock()

		// Accept a new connection
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("Accept error")
			continue
		}
		fmt.Println("New connection: ", conn.RemoteAddr())

		// Lock the mutex to safely update the connection count
		s.mutex.Lock()
		s.connectionCount++
		// Unlock the mutex after updating the connection count
		s.mutex.Unlock()

		// Handle the new connection in a goroutine
		go s.handleClient(conn)
	}
}

func (s *Server) DisconnectClient(conn net.Conn, clientName string) {
	// Lock the mutex to safely update the connection count
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Close the client's connection
	conn.Close()

	// Remove the client from the map
	delete(s.clients, conn)

	// Update the connection count
	s.connectionCount--

	// Broadcast a message to inform others about the disconnection
	s.broadcastJoinDisc(clientName, clientName+" has left our chat...")
}

func (s *Server) handleClient(conn net.Conn) {
	defer func() {
		// Decrement the connection count when the client disconnects
		s.mutex.Lock()
		s.connectionCount--
		s.mutex.Unlock()

		conn.Close()
	}()

	// Prompt the client for their name
	s.sendMessage(conn, "Welcome to TCP-Chat!\n")
	s.sendMessage(conn, logo+"\n")

	var name string
	var err error

	for{
		s.sendMessage(conn, "[ENTER YOUR NAME]:")
		name, err = s.receiveMessage(conn)
		if err != nil {
			fmt.Println("Error receiving name:", err)
			return
		}
		if strings.TrimSpace(name) != ""{
			break
		}
	}
	// s.sendMessage(conn, "[ENTER YOUR NAME]:")
	// name, err := s.receiveMessage(conn)

	// if err != nil {
	// 	fmt.Println("Error receiving name:", err)
	// 	return
	// }

	clientName := strings.TrimSpace(name)

	// Add the client to the map
	s.mutex.Lock()
	s.clients[conn] = clientName
	s.mutex.Unlock()

	s.broadcastJoinDisc(clientName, clientName+" has joined our chat...")

	s.sendMessagesToClient(conn)

	for {
		s.sendMessage(conn, fmt.Sprintf("[%s][%s]:", time.Now().Format("2006-01-02 15:04:05"), clientName))
		// Read the message from the client
		message, err := s.receiveMessage(conn)
		if err != nil {
			// fmt.Printf("%s has left the chat\n", clientName)
			s.broadcastJoinDisc(clientName, clientName+" has left our chat...")
			break
		}

		// s.msgch <- Message{
		// 	from:    clientName,
		// 	payload: message,
		// }
		if message != "" {
			s.messageBuffer = append(s.messageBuffer, Message{clientName, message, time.Now().Format("2006-01-02 15:04:05")})
			s.broadcastMessage(clientName, message)
		}
	}
}

func (s *Server) sendMessage(conn net.Conn, message string) {
	conn.Write([]byte(message))
}

func (s *Server) receiveMessage(conn net.Conn) (string, error) {
	message, err := bufio.NewReader(conn).ReadString('\n')
	return strings.TrimSpace(message), err
}

// Broadcast the message to all other clients
func (s *Server) broadcastMessage(from, message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	formattedMessage := fmt.Sprintf("\n[%s][%s]:%s", time.Now().Format("2006-01-02 15:04:05"), from, message)

	for otherConn, otherName := range s.clients {
		if otherName != from {
			s.sendMessage(otherConn, formattedMessage)
			s.sendMessage(otherConn, fmt.Sprintf("\n[%s][%s]:", time.Now().Format("2006-01-02 15:04:05"), otherName))
		}
	}
}

func (s *Server) broadcastJoinDisc(from, message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for otherConn, otherName := range s.clients {
		if otherName != from {
			s.sendMessage(otherConn, "\n"+message)
			s.sendMessage(otherConn, fmt.Sprintf("\n[%s][%s]:", time.Now().Format("2006-01-02 15:04:05"), otherName))
		}
	}
}

func main() {
	args := os.Args[1:]
	if len(args) <= 1 {
		port := "8989"
		if len(args) == 1 {
			if !(IsPortValid(args[0])) {
				fmt.Println("[USAGE]: ./TCPChat $port")
				os.Exit(0)
			}
			port = args[0]
		}
		server := NewServer(":" + port)
		fmt.Printf("Listening on the port :%s\n", port)


		if err := server.Start(); err != nil {
			fmt.Println("Error starting server:", err)
			fmt.Println("[USAGE]: ./TCPChat $port")
			os.Exit(0)
		}
	} else {
		fmt.Println("[USAGE]: ./TCPChat $port")
		os.Exit(0)
	}
}

func IsInt(value string) bool {
	if value == "" {
		return false
	}

	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}



func IsPortValid(port string) bool {
	if !IsInt(port) {
		return false
	}

	portNumber := atoi(port)
	return portNumber >= 1 && portNumber <= 65535
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}