# net-cat

## Description
Net-Cat is a simple replication of the net-cat utility tool. Where a server listens on a specified port by the user or the default (8989) and accepts incoming client connections that allow them to join a group chat.

## Details
- The project is written using Golang
- Accepts a port number at runtime or uses the default port 8989
- Clients are required to enter their name to enter the group chat
- Clients will be able to send and receive messages as well as view the chat history upon joining
- All clients are notified when a client joins or leaves the chat
- Maximum of 10 clients


## Usage: How to run?
- Clone the repository:
- Navigate to the correct directory: cd net-cat
- To start the server: go run TCPChat.go [port]
- To connect as client: nc $IP $port
- Ex. nc localhost 8989
