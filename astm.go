package lis1a2

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/therealriteshkudalkar/lis1a2/connection"
	"github.com/therealriteshkudalkar/lis1a2/constants"
)

type ASTMConnection struct {
	connection                connection.Connection
	incomingMessage           chan string
	status                    constants.LIS1A2ConnectionStatus
	frameNumber               int
	ackChan                   chan bool
	buffer                    []byte
	recordBuffer              string
	messageBuffer             string
	numberOfConnectionRetries int
	internalCtx               context.Context
	internalCtxCancelFunc     context.CancelFunc
	saveIncomingMessage       bool
	incomingMessageSaveDir    string
}

func NewASTMConnection(conn connection.Connection, saveIncomingMessage bool, incomingMessageSaveDir ...string) *ASTMConnection {
	astmConn := &ASTMConnection{
		connection:                conn,
		status:                    constants.Idle,
		buffer:                    make([]byte, 0),
		recordBuffer:              "",
		messageBuffer:             "",
		frameNumber:               0,
		numberOfConnectionRetries: 0,
	}
	if saveIncomingMessage && len(incomingMessageSaveDir) > 0 {
		astmConn.saveIncomingMessage = true
		astmConn.incomingMessageSaveDir = incomingMessageSaveDir[0]
	} else {
		astmConn.saveIncomingMessage = false
	}
	return astmConn
}

// Connect runs connect method of underlying Connection object
func (astmConn *ASTMConnection) Connect() error {
	astmConn.numberOfConnectionRetries = 0
	err := astmConn.connection.Connect()
	for err != nil {
		astmConn.numberOfConnectionRetries += 1
		if astmConn.numberOfConnectionRetries > constants.MaxConnectionRetires {
			return err
		}
		err = astmConn.connection.Connect()
	}
	astmConn.internalCtx, astmConn.internalCtxCancelFunc = context.WithCancel(context.Background())
	astmConn.ackChan = make(chan bool, 1)
	astmConn.incomingMessage = make(chan string, 1)
	return nil
}

// Disconnect runs the disconnect method of underlying Connection object and also closes channels
func (astmConn *ASTMConnection) Disconnect() error {
	close(astmConn.incomingMessage)
	close(astmConn.ackChan)
	if err := (astmConn.connection).Disconnect(); err != nil {
		return err
	}
	return nil
}

func (astmConn *ASTMConnection) IsConnected() bool {
	return astmConn.connection.IsConnected()
}

func (astmConn *ASTMConnection) ChangeStatus(status constants.LIS1A2ConnectionStatus) {
	astmConn.status = status
}

func (astmConn *ASTMConnection) WaitForACK() bool {
	timerInterrupt := time.NewTimer(time.Second * 15)
	select {
	case resp, ok := <-astmConn.ackChan:
		if !ok {
			slog.Error("Disconnected while waiting for ACK.")
			return false
		}
		slog.Debug("ACK/NAK received.", "Type", resp)
		if !timerInterrupt.Stop() {
			slog.Debug("Draining the timer channel for WaitForACK.")
			<-timerInterrupt.C
			slog.Debug("Drained the timer channel for WaitForACK.")
		}
		slog.Debug("Stopped the timer.")
		return resp
	case <-timerInterrupt.C:
		slog.Debug("Timer interrupt for WaitForACK.")
		return false
	}
}

func (astmConn *ASTMConnection) StopSendMode() {
	data := string([]byte{constants.EOT})
	(astmConn.connection).Write(data)
	slog.Debug("Sending EOT.")
	astmConn.status = constants.Idle
	slog.Debug("Changed mode to Idle and stopped send mode.")
}

func (astmConn *ASTMConnection) EstablishSendMode() bool {
	astmConn.frameNumber = 1
	if astmConn.status != constants.Idle {
		slog.Error("Connection not in idle when trying to establish send mode.")
		return false
	}
	astmConn.status = constants.Establishing
	slog.Debug("Establishing send mode.")
	(astmConn.connection).Write(string([]byte{constants.ENQ}))
	slog.Debug("Sent ENQ.")
	if !astmConn.WaitForACK() {
		slog.Error("Could not establish send mode.")
		astmConn.StopSendMode()
		return false
	}
	astmConn.status = constants.Sending
	slog.Debug("Changing status to sending.")
	return true
}

// ReadMessage reads a single ASTM Message from the connection.
func (astmConn *ASTMConnection) ReadMessage(timeout time.Duration) (error, string) {
	timerInterrupt := time.NewTimer(timeout)
	select {
	case newMessage, ok := <-astmConn.incomingMessage:
		if !ok {
			return errors.New("channel closed while reading"), ""
		}
		slog.Debug("New astm message arrived.")
		if !timerInterrupt.Stop() {
			slog.Debug("Draining timer channel for ReadMessage.")
			<-timerInterrupt.C
			slog.Debug("Drained timer channel for ReadMessage.")
		}
		slog.Debug("Stopped timer!")
		return nil, newMessage
	case <-timerInterrupt.C:
		slog.Debug("Timer interrupt in ReadMessage.")
		return errors.New("read message timer timed out"), ""
	}
}

func (astmConn *ASTMConnection) SaveIncomingMessage(message string, fileDir string) {
	currentTime := time.Now()
	timeStamp := currentTime.Format("2006-01-02-15-04-05")
	var filePath string
	if strings.HasSuffix(fileDir, "/") {
		filePath = fmt.Sprintf("%v%v.txt", fileDir, timeStamp)
	} else {
		filePath = fmt.Sprintf("%v/%v.txt", fileDir, timeStamp)
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		slog.Error("Error while creating a file.", "Error", err)
		return
	}
	slog.Debug("File created for query message.", "File", filePath)
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			slog.Error("Error while closing file.", "Error", err)
		}
	}(file)

	formattedMessage := fmt.Sprintf("Timestamp: %v\n", timeStamp) +
		fmt.Sprintf("Bytes Array: \n%v\n", []byte(message)) +
		fmt.Sprintf("Message in string: \n%v\n", message)

	writeCount, err := file.Write([]byte(formattedMessage))
	if err != nil {
		slog.Error("Error while writing to the file.", "Error", err)
		return
	}
	slog.Debug("Bytes written to file.", "Write count", writeCount)
}

func (astmConn *ASTMConnection) CalculateChecksum(frame string) []byte {
	byteFrame := []byte(frame)
	var sum = 0
	for _, bt := range byteFrame {
		sum = (sum + int(bt)) % 256
	}
	calcChecksum := strings.ToUpper(hex.EncodeToString([]byte{byte(sum)}))
	calcChecksumBytes := []byte(calcChecksum)
	slog.Debug("Calculate Checksum.", "Checksum:", calcChecksum, "In bytes: ", calcChecksumBytes)
	return calcChecksumBytes
}

func (astmConn *ASTMConnection) IsFrameValid(frame string) bool {
	byteFrame := []byte(frame)
	frameLen := len(byteFrame)
	etxOrEtb := byteFrame[frameLen-5]
	if frameLen < 5 || byteFrame[0] != constants.STX || byteFrame[frameLen-1] != constants.LF ||
		byteFrame[frameLen-2] != constants.CR || (etxOrEtb != constants.ETX && etxOrEtb != constants.ETB) ||
		byteFrame[frameLen-6] != constants.CR {
		return false
	}
	return true
}

func (astmConn *ASTMConnection) IsTheFrameIntermediate(frame string) bool {
	byteFrame := []byte(frame)
	byteFrameLen := len(byteFrame)
	isIntermediate := byteFrame[byteFrameLen-5] == constants.ETB
	slog.Debug("Checking frame type.", "Is it intermediate", isIntermediate)
	return isIntermediate
}

func (astmConn *ASTMConnection) CheckChecksum(frame string) bool {
	if !astmConn.IsFrameValid(frame) {
		slog.Error("Checking checksum. Given frame is invalid.")
		return false
	}
	byteFrame := []byte(frame)
	frameLen := len(byteFrame)
	calculatedChecksum := astmConn.CalculateChecksum(string(byteFrame[1 : frameLen-4]))
	receivedChecksum := byteFrame[frameLen-4 : frameLen-2]
	doesCheckSumMatch := bytes.Equal(receivedChecksum, calculatedChecksum)
	slog.Debug("Checking checksum.", "Received", receivedChecksum, "Calculated", calculatedChecksum)
	return doesCheckSumMatch
}

func (astmConn *ASTMConnection) sendString(frame string) {
	if astmConn.status != constants.Sending {
		slog.Error("Connection not in send mode when trying to send data.")
		return
	}
	var byteArr []byte
	byteArr = append(byteArr, constants.STX)
	byteArr = append(byteArr, []byte(frame)...)
	byteArr = append(byteArr, astmConn.CalculateChecksum(frame)...)
	byteArr = append(byteArr, constants.CR)
	byteArr = append(byteArr, constants.LF)
	tmpSendStr := string(byteArr)
	(astmConn.connection).Write(tmpSendStr)
	tryCounter := 0
	for !astmConn.WaitForACK() {
		tryCounter++
		if tryCounter > 5 {
			astmConn.StopSendMode()
			slog.Error("Max number of send retires reached.")
			return
		}
		(astmConn.connection).Write(tmpSendStr)
	}
	slog.Debug("Frame sent successfully.")
}

func (astmConn *ASTMConnection) sendEndFrame(frameNumber int, frame string) {
	slog.Debug("Sending ending frame with ETX.")
	var byteArr []byte
	hexFrameNumber := hex.EncodeToString([]byte{byte(frameNumber)})[1:]
	byteArr = append(byteArr, []byte(hexFrameNumber)...)
	byteArr = append(byteArr, []byte(frame)...)
	byteArr = append(byteArr, constants.CR)
	byteArr = append(byteArr, constants.ETX)
	astmConn.sendString(string(byteArr))
}

func (astmConn *ASTMConnection) sendIntermediateFrame(frameNumber int, frame string) {
	slog.Debug("Sending intermediate frame with ETB.")
	var byteArr []byte
	hexFrameNumber := hex.EncodeToString([]byte{byte(frameNumber)})[1:]
	byteArr = append(byteArr, []byte(hexFrameNumber)...)
	byteArr = append(byteArr, []byte(frame)...)
	byteArr = append(byteArr, constants.ETB)
	astmConn.sendString(string(byteArr))
}

// SendMessage takes single ASTM Record as input and sends it as one or more frames over the connection
func (astmConn *ASTMConnection) SendMessage(message string) {
	byteMessage := []byte(message)
	for len(byteMessage) > constants.MaxFrameSize {
		// divide it in chunks
		intermediateFrame := string(byteMessage[:constants.MaxFrameSize])
		astmConn.sendIntermediateFrame(astmConn.frameNumber, intermediateFrame)
		astmConn.frameNumber = (astmConn.frameNumber + 1) % 8
		byteMessage = byteMessage[constants.MaxFrameSize:]
	}
	astmConn.sendEndFrame(astmConn.frameNumber, string(byteMessage))
	astmConn.frameNumber = (astmConn.frameNumber + 1) % 8
}

func (astmConn *ASTMConnection) connectionDataReceived(data string) {
	byteData := []byte(data)
	lenOfData := len(byteData)

	slog.Debug("Byte data arrived.", "Data", byteData)
	slog.Debug("Current status of Automaton.", "State", astmConn.status)

	if lenOfData > 0 {
		for _, singleByte := range byteData {
			switch astmConn.status {
			case constants.Idle:
				if singleByte != constants.ENQ {
					(astmConn.connection).Write(string([]byte{constants.NAK}))
				} else {
					slog.Info("Received ENQ in Idle state. Sending ACK.")
					(astmConn.connection).Write(string([]byte{constants.ACK}))
					astmConn.status = constants.Receiving
					// TODO: Change it back to idle if nothing is received even after 15 seconds have passed
				}
			case constants.Sending:
				receivedACK := singleByte == constants.ACK
				slog.Debug("Waiting for ACK in sending state.")
				astmConn.ackChan <- receivedACK
				slog.Debug("Received.", "ACK type", receivedACK)
			case constants.Receiving:
				if singleByte == constants.ENQ {
					(astmConn.connection).Write(string([]byte{constants.NAK}))
				} else if singleByte != constants.EOT {
					if singleByte != constants.NUL {
						astmConn.buffer = append(astmConn.buffer, singleByte)
					}
					if singleByte == constants.LF {
						receivedFrame := string(astmConn.buffer)
						receivedFrameLen := len(receivedFrame)
						astmConn.buffer = make([]byte, 0)
						if !astmConn.CheckChecksum(receivedFrame) {
							slog.Error("Checksum did not match. Sending NAK.")
							(astmConn.connection).Write(string([]byte{constants.NAK}))
						} else {
							slog.Debug("Checksum ok. Sending ACK.")
							(astmConn.connection).Write(string([]byte{constants.ACK}))
							if astmConn.IsTheFrameIntermediate(receivedFrame) {
								partialRecord := receivedFrame[2 : receivedFrameLen-5]
								astmConn.recordBuffer += partialRecord
							} else {
								partialRecord := receivedFrame[2 : receivedFrameLen-6]
								astmConn.recordBuffer += partialRecord
								astmConn.messageBuffer += astmConn.recordBuffer + "\n"
								astmConn.recordBuffer = ""
							}
						}
					}
				} else {
					slog.Debug("Received EOT in Receiving state. Going to Idle state.")
					if len(astmConn.messageBuffer) != 0 {
						if astmConn.saveIncomingMessage {
							go astmConn.SaveIncomingMessage(astmConn.messageBuffer, astmConn.incomingMessageSaveDir)
						}
						astmConn.incomingMessage <- astmConn.messageBuffer
						astmConn.messageBuffer = ""
					}
					astmConn.status = constants.Idle
					slog.Debug("State changed to Idle.")
					return
				}
			case constants.Establishing:
				if singleByte == constants.ACK {
					slog.Debug("Received ACK in Establishing state.")
					astmConn.ackChan <- true
					return
				} else if singleByte == constants.ENQ {
					slog.Debug("Received ENQ in Establishing state.")
					time.Sleep(time.Second * 1)
					(astmConn.connection).Write(string([]byte{constants.ENQ}))
					slog.Debug("Sent ENQ.")
					return
				} else {
					continue
				}
			default:
				slog.Error("In incorrect state.", "Skipping byte", singleByte)
			}
		}
	}
}

// Listen listens to the incoming messages over the connection
func (astmConn *ASTMConnection) Listen() {
	(astmConn.connection).Listen()
	for {
		str, err := (astmConn.connection).ReadStringFromConnection()
		if err != nil {
			slog.Error("Stopped listening.", "Error", err)
			return
		}
		astmConn.connectionDataReceived(str)
		select {
		case <-astmConn.internalCtx.Done():
			slog.Debug("Ceasing Listen operation on ASTM connection.")
			return
		default:
			continue
		}
	}
}
