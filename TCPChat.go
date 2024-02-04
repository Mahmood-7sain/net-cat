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

type Server struct {
	listenAddr      string   //Port to listen on
	ln              net.Listener  //The actual listener
	quitch          chan struct{}  //To stop the server
	msgch           chan Message   //Channel to send messages to the clients
	clients         map[net.Conn]string // Added a map to store connected clients and their names
	mutex           sync.Mutex      // Mutex to protect from concurrent access
	connectionCount int             // Count of connected clients
	messageBuffer   []Message       // Buffer to store messages received from clients
}

type Message struct {
	from    string // The name of the client that sent the message
	payload string // The message payload
	time    string // The time the message was sent
}

//Creates a new TCP server with the given listen address.
func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		msgch:      make(chan Message, 10),
		clients:    make(map[net.Conn]string),
	}
}




//Starts the TCP server and listens for incoming connections
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


//Loops and accepts new connections 
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


//Handles new client connections and continues to read msgs
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
	s.sendMessage(conn, logo+"\n") //Prints linux logo

	var name string
	var err error

	//Loop until the client inputs a name
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

	clientName := strings.TrimSpace(name)

	// Add the client to the map
	s.mutex.Lock()
	s.clients[conn] = clientName
	s.mutex.Unlock()

	s.broadcastJoinDisc(clientName, clientName+" has joined our chat...")

	s.sendMessagesToClient(conn) //Send all previous msgs to the new client

	for {
		s.sendMessage(conn, fmt.Sprintf("[%s][%s]:", time.Now().Format("2006-01-02 15:04:05"), clientName))
		// Read the message from the client
		message, err := s.receiveMessage(conn)
		if err != nil {
			s.broadcastJoinDisc(clientName, clientName+" has left our chat...")
			break
		}

		//Only accept non empty msgs
		if message != "" {
			s.messageBuffer = append(s.messageBuffer, Message{clientName, message, time.Now().Format("2006-01-02 15:04:05")})
			s.broadcastMessage(clientName, message)
		}
	}
}

//Writes the msg to the passes connection(client)
func (s *Server) sendMessage(conn net.Conn, message string) {
	conn.Write([]byte(message))
}


//Receives a message from the client and returns it as a string.
func (s *Server) receiveMessage(conn net.Conn) (string, error) {
	message, err := bufio.NewReader(conn).ReadString('\n')
	return strings.TrimSpace(message), err
}

//Sends all previous msgs stored in the buffer to the new client
func (s *Server) sendMessagesToClient(conn net.Conn) {
	for _, msg := range s.messageBuffer {
		s.sendMessage(conn, fmt.Sprintf("[%s][%s]:%s\n", msg.time, msg.from, msg.payload))
	}
}


// Broadcasts the message to all clients except the sender
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


//Brodcasts to all connected clients when a client joins or leaves the chat
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


//Takes the port number as an argument and starts the server on that port.
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
		server := NewServer("0.0.0.0:" + port)
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


//Checks if a passed string only contains valid integers
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


//Checks whether the given port is valid or not
func IsPortValid(port string) bool {
	if !IsInt(port) {
		return false
	}

	portNumber := atoi(port)
	return portNumber >= 1 && portNumber <= 65535
}

//Returns a passed string as an int
func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}