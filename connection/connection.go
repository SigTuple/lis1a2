package connection

type Connection interface {
	Connect() error
	IsConnected() bool
	Listen()
	Write(data []byte)
	ReadStringFromConnection() (string, error)
	Disconnect() error
}
