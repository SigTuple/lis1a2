package tests

import (
	"github.com/therealriteshkudalkar/lis1a2"
	"github.com/therealriteshkudalkar/lis1a2/connection"
	"log"
)

func testTCPConnectDisconnect() {
	var tcpConn = connection.NewTCPConnection("localhost", "4000")
	if err := tcpConn.Connect(); err != nil {
		log.Fatalf("Failed to connect to TCP server.")
	}
	defer func() {
		if err := tcpConn.Disconnect(); err != nil {
			log.Printf("Failed to disconnect from the TCP server.")
		}
	}()

	astmConn := lis1a2.NewASTMConnection(&tcpConn)
	err := astmConn.Connect()
	if err != nil {
		return
	}
}
