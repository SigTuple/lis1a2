# LIS1A2 - A Go library

Standard implementation for the lis1a2 protocol. This library is extensible,
so instead of using a TCP connection, the user can use any other standard for 
communication like RS232 provided it adheres to the `Connection` interface provided
in the library.

## Installation

```bash
go get github.com/therealriteshkudalkar/lis1a2
```

## Features

- Adheres to LIS1A2 Standard
- Implementation for TCP Connection adhering to `Connection` interface is provided.

## Usage

The user needs to initialize the Connection object. We'll use the TCP implementation
bundled with the library.
The user needs to use the connection object to create a ASTM Connection.

```go
package main

import (
	"log"

	"github.com/therealriteshkudalkar/lis1a2"
	"github.com/therealriteshkudalkar/lis1a2/connection"
)

func main() {
	var tcpConn = connection.NewTCPConnection("localhost", "4000")

	var astmConn = lis1a2.NewASTMConnection(&tcpConn, false)
	err := astmConn.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to the ASTM Service")
	}
	defer func(astmConn *lis1a2.ASTMConnection) {
		err := astmConn.Disconnect()
		if err != nil {

		}
	}(astmConn)
}
```

To send a new message to the astm connection the user needs to establish send mode

```go
package main

import (
	"log"
	
	"github.com/therealriteshkudalkar/lis1a2"
)

func SendMessageToConnection(astmConn lis1a2.ASTMConnection, data []byte) {
	establishedSendMode := astmConn.EstablishSendMode()
	if !establishedSendMode {
		log.Fatal("Could not establish send mode.")
	}
	astmConn.SendMessage(data)
	astmConn.StopSendMode()
}
```




